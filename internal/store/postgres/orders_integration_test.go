package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"os"
	"sync"
	"testing"
	"time"
)

type orderFixture struct {
	pool                      *pgxpool.Pool
	manager                   *service.Orders
	actor, customer, kg, item int64
}

func setupOrders(t *testing.T) orderFixture {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL")
	}
	ctx := context.Background()
	pool, err := Open(ctx, url, 5*time.Second)
	if err != nil {
		t.Fatal("connect QA database")
	}
	f := orderFixture{pool: pool, manager: service.NewOrders(NewOrderStore(pool))}
	suffix := fmt.Sprint(time.Now().UnixNano())
	if err := pool.QueryRow(ctx, `INSERT INTO laundry.users(name,username,password_hash,role) VALUES('Order tester',$1,'unused','cashier') RETURNING id`, "orders_"+suffix).Scan(&f.actor); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO laundry.customers(name,whatsapp,whatsapp_opt_in) VALUES('Order customer','+6281234567890',true) RETURNING id`).Scan(&f.customer); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO laundry.services(name,unit,price,estimated_hours) VALUES('Cuci KG','KG',8000,48) RETURNING id`).Scan(&f.kg); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO laundry.services(name,unit,price,estimated_hours) VALUES('Selimut','ITEM',25000,72) RETURNING id`).Scan(&f.item); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM laundry.orders WHERE created_by=$1`, f.actor)
		pool.Exec(ctx, `DELETE FROM laundry.customers WHERE id=$1`, f.customer)
		pool.Exec(ctx, `DELETE FROM laundry.services WHERE id=ANY($1)`, []int64{f.kg, f.item})
		pool.Exec(ctx, `DELETE FROM laundry.users WHERE id=$1`, f.actor)
		pool.Close()
	})
	return f
}
func (f orderFixture) input(key string) service.OrderInput {
	return service.OrderInput{CustomerID: f.customer, DueAt: time.Now().Add(48 * time.Hour).Truncate(time.Second), Notes: "Tanpa pewangi", Items: []service.OrderItemInput{{ServiceID: f.kg, Quantity: "4"}}, RequestKey: key}
}
func TestOrderCreationPostgres(t *testing.T) {
	f := setupOrders(t)
	ctx := context.Background()
	input := f.input("creation_test_key_123456")
	order, err := f.manager.Create(ctx, f.actor, input)
	if err != nil {
		t.Fatal(err)
	}
	if order.Total != 32000 || order.Status != "RECEIVED" || order.PaymentStatus != "UNPAID" || len(order.Items) != 1 || order.Items[0].PriceVersionID == 0 {
		t.Fatal("order defaults/snapshot incorrect")
	}
	replay, err := f.manager.Create(ctx, f.actor, input)
	if err != nil || replay.ID != order.ID {
		t.Fatal("retry duplicated order", err)
	}
	input.Notes = "Changed"
	if _, err := f.manager.Create(ctx, f.actor, input); err == nil {
		t.Fatal("idempotency key reused for changed data")
	}
	if _, err := f.pool.Exec(ctx, `UPDATE laundry.services SET price=12000,name='Changed name' WHERE id=$1`, f.kg); err != nil {
		t.Fatal(err)
	}
	snapshot, err := f.manager.Get(ctx, order.ID)
	if err != nil || snapshot.Total != 32000 || snapshot.Items[0].UnitPrice != 8000 || snapshot.Items[0].ServiceName != "Cuci KG" {
		t.Fatal("master update changed old transaction", err)
	}
	input.RequestKey = "inactive_test_key_123456"
	f.pool.Exec(ctx, `UPDATE laundry.services SET is_active=false WHERE id=$1`, f.kg)
	if _, err := f.manager.Create(ctx, f.actor, input); err == nil {
		t.Fatal("inactive service accepted")
	}
	var name string
	f.pool.QueryRow(ctx, `SELECT name FROM laundry.customers WHERE id=$1`, f.customer).Scan(&name)
	if name != "Order customer" {
		t.Fatal("customer was silently changed")
	}
	var count int
	f.pool.QueryRow(ctx, `SELECT count(*) FROM laundry.orders WHERE created_by=$1`, f.actor).Scan(&count)
	if count != 1 {
		t.Fatal("failed request persisted partial order")
	}
}

func TestMultipleOrderServicesPostgres(t *testing.T) {
	f := setupOrders(t)
	ctx := context.Background()
	input := f.input("multiple_services_123456")
	input.Items = []service.OrderItemInput{{ServiceID: f.kg, Quantity: "4.125"}, {ServiceID: f.item, Quantity: "2"}}
	order, err := f.manager.Create(ctx, f.actor, input)
	if err != nil || len(order.Items) != 2 || order.Total != 83000 || order.Items[0].QuantityMilli != 4125 || order.Items[1].Unit != "ITEM" {
		t.Fatal("multi-service calculation failed", err)
	}
	input.RequestKey = "invalid_item_count_123456"
	input.Items[1].Quantity = "1.5"
	if _, err := f.manager.Create(ctx, f.actor, input); err == nil {
		t.Fatal("fractional pcs accepted")
	}
	var count int
	f.pool.QueryRow(ctx, `SELECT count(*) FROM laundry.orders WHERE created_by=$1`, f.actor).Scan(&count)
	if count != 1 {
		t.Fatal("invalid second line left partial transaction")
	}
	input.Items = nil
	input.RequestKey = "empty_order_items_123456"
	if _, err := f.manager.Create(ctx, f.actor, input); err == nil {
		t.Fatal("empty order accepted")
	}
}

func TestServerVerifiesPresentedOrderTotal(t *testing.T) {
	f := setupOrders(t)
	ctx := context.Background()
	input := f.input("verify_order_total_12345")
	expected := int64(31000)
	input.ExpectedTotal = &expected
	if _, err := f.manager.Create(ctx, f.actor, input); err == nil {
		t.Fatal("client total mismatch accepted")
	}
	var count int
	f.pool.QueryRow(ctx, `SELECT count(*) FROM laundry.orders WHERE created_by=$1`, f.actor).Scan(&count)
	if count != 0 {
		t.Fatal("mismatched total persisted")
	}
	expected = 32000
	order, err := f.manager.Create(ctx, f.actor, input)
	if err != nil || order.Total != expected {
		t.Fatal("matching total rejected", err)
	}
}

func TestConcurrentUniqueOrderNumbersAndRetries(t *testing.T) {
	f := setupOrders(t)
	ctx := context.Background()
	base := f.input("concurrent_order_123456")
	type result struct {
		order service.Order
		err   error
	}
	results := make(chan result, 12)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			input := base
			input.RequestKey = fmt.Sprintf("concurrent_number_%016d", i)
			o, err := f.manager.Create(ctx, f.actor, input)
			results <- result{o, err}
		}(i)
	}
	wg.Wait()
	close(results)
	seen := map[string]bool{}
	for r := range results {
		if r.err != nil {
			t.Fatal(r.err)
		}
		if seen[r.order.Number] || r.order.Number == "" {
			t.Fatal("duplicate order number")
		}
		seen[r.order.Number] = true
	}
	results = make(chan result, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); o, err := f.manager.Create(ctx, f.actor, base); results <- result{o, err} }()
	}
	wg.Wait()
	close(results)
	var id int64
	for r := range results {
		if r.err != nil {
			t.Fatal(r.err)
		}
		if id != 0 && r.order.ID != id {
			t.Fatal("concurrent retry created another order")
		}
		id = r.order.ID
	}
}

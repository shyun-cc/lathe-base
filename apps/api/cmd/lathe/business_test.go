package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/shyun-cc/lathe-base/apps/api/internal/module/printing"
)

type fields map[string]any

type reply struct {
	status  int
	Code    int             `json:"code"`
	TraceID string          `json:"traceId"`
	Data    json.RawMessage `json:"data"`
	raw     []byte
	err     error
}

type testAPI struct {
	url, auth string
	client    *http.Client
}

func startAPI(t *testing.T, db *sql.DB, token string) testAPI {
	t.Helper()
	server := ghttp.GetServer(t.Name())
	server.SetAddr("127.0.0.1:0")
	server.SetDumpRouterMap(false)
	bindBusiness(server, db, token)
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.Shutdown(); err != nil {
			t.Error(err)
		}
	})
	return testAPI{fmt.Sprintf("http://127.0.0.1:%d/api/v1", server.GetListenedPort()), "Bearer " + token,
		&http.Client{Timeout: 15 * time.Second}}
}

func (a testAPI) send(method, path string, body any) reply {
	var data []byte
	switch value := body.(type) {
	case nil:
	case json.RawMessage:
		data = value
	default:
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			return reply{err: err}
		}
	}
	req, err := http.NewRequest(method, a.url+path, bytes.NewReader(data))
	if err != nil {
		return reply{err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", a.auth)
	resp, err := a.client.Do(req)
	if err != nil {
		return reply{err: err}
	}
	defer resp.Body.Close()
	result := reply{status: resp.StatusCode}
	result.raw, result.err = io.ReadAll(resp.Body)
	if result.err == nil {
		result.err = json.Unmarshal(result.raw, &result)
	}
	return result
}

func requireReply(t *testing.T, result reply, want int) {
	t.Helper()
	if result.err != nil || result.status != want || result.TraceID == "" ||
		(want == 200 && result.Code != 0) || (want != 200 && result.Code != want) {
		t.Fatalf("HTTP %d, want %d, body %s, error %v", result.status, want, result.raw, result.err)
	}
}

func (a testAPI) call(t *testing.T, method, path string, body any, want int) reply {
	t.Helper()
	result := a.send(method, path, body)
	t.Logf("%s %s returned HTTP %d", method, path, result.status)
	requireReply(t, result, want)
	return result
}

func readData[T any](t *testing.T, r reply) T {
	t.Helper()
	var data T
	if err := json.Unmarshal(r.Data, &data); err != nil {
		t.Fatal(err)
	}
	return data
}

type idResult struct {
	ID int64 `json:"id"`
}

func (a testAPI) create(t *testing.T, path string, body any) int64 {
	t.Helper()
	id := readData[idResult](t, a.call(t, "POST", path, body, 200)).ID
	if id <= 0 {
		t.Fatalf("missing id for %s", path)
	}
	return id
}

func pathID(path string, id int64) string { return fmt.Sprintf("%s/%d", path, id) }

func TestBusinessRequestValidation(t *testing.T) {
	token := randomHex()
	api := startAPI(t, nil, token)
	unauthorized := api
	unauthorized.auth = "Bearer invalid"
	unauthorized.call(t, "GET", "/parties", nil, 401)
	unauthorized.call(t, "GET", "/session", nil, 401)
	unauthorized.call(t, "GET", "/print-documents/sales-order/1", nil, 401)
	api.call(t, "GET", "/session", nil, 200)
	api.call(t, "GET", "/print-documents/unknown/1", nil, 400)
	api.call(t, "GET", "/parties?limit=201", nil, 400)
	api.call(t, "GET", "/parties/0?id=3", nil, 400)
	api.call(t, "POST", "/parties", json.RawMessage(`{"code":"A","name":"a","unknown":true}`), 400)
	api.call(t, "POST", "/parties", json.RawMessage(`{} {}`), 400)
	api.call(t, "POST", "/parties", fields{"code": "A", "name": "a"}, 400)
}

func randomHex() string {
	var data [16]byte
	rand.Read(data[:])
	return hex.EncodeToString(data[:])
}

func isolatedPostgres(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("LATHE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("LATHE_TEST_DATABASE_URL is required for the PostgreSQL 18 integration test")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid LATHE_TEST_DATABASE_URL")
	}
	admin := stdlib.OpenDB(*config)
	t.Cleanup(func() { admin.Close() })
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	var version int
	if err := admin.QueryRowContext(ctx, "SELECT current_setting('server_version_num')::int").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version/10000 != 18 {
		t.Fatalf("PostgreSQL 18 required, got version number %d", version)
	}
	database := "lathe_test_" + randomHex()
	identifier := pgx.Identifier{database}.Sanitize()
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+identifier+" TEMPLATE template0"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(ctx, "DROP DATABASE "+identifier); err != nil {
			t.Error(err)
		}
	})
	config.Database = database
	db := stdlib.OpenDB(*config)
	t.Cleanup(func() { db.Close() })
	for range 2 {
		if err := migrateUp(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestPostgresBusinessWorkflow(t *testing.T) {
	db := isolatedPostgres(t)
	api := startAPI(t, db, randomHex())
	customer := api.create(t, "/parties", fields{"code": "C01", "name": "Example Customer", "is_customer": true})
	supplier := api.create(t, "/parties", fields{"code": "S01", "name": "Example Supplier", "is_supplier": true})
	unit := api.create(t, "/units", fields{"code": "PCS", "name": "Piece"})
	product := api.create(t, "/products", fields{"code": "P01", "name": "Example Part", "unit_id": unit})
	turning := api.create(t, "/operations", fields{"code": "TURN", "name": "Turning"})
	finishing := api.create(t, "/operations", fields{"code": "FINISH", "name": "Finishing"})
	employee := api.create(t, "/employees", fields{"code": "E01", "name": "Example Operator"})
	employeePath := pathID("/employees", employee)
	api.call(t, "PUT", employeePath, fields{"code": "E01", "name": "Operator One"}, 200)
	list := readData[[]fields](t, api.call(t, "GET", "/employees?limit=1", nil, 200))
	if len(list) != 1 || list[0]["name"] != "Operator One" {
		t.Fatalf("employee list = %v", list)
	}
	spare := api.create(t, "/employees", fields{"code": "E02", "name": "Example Temporary Operator"})
	api.call(t, "DELETE", pathID("/employees", spare), nil, 200)
	api.call(t, "GET", pathID("/employees", spare), nil, 404)

	orderRequest := fields{"order_no": "SO-1", "customer_id": customer, "currency": "CNY", "ordered_on": "2026-01-01"}
	line := fields{"product_id": product, "quantity_milli": 10000, "amount_minor": 10001}
	orderRequest["lines"] = []fields{line, {"product_id": -1, "quantity_milli": 1000, "amount_minor": 200}}
	api.call(t, "POST", "/sales-orders", orderRequest, 400)
	orderRequest["lines"] = []fields{line}
	order := api.create(t, "/sales-orders", orderRequest) // The failed multi-line request released its document number.
	orderPath := pathID("/sales-orders", order)
	type orderDetail struct {
		Lines []idResult `json:"lines"`
	}
	orderData := readData[orderDetail](t, api.call(t, "GET", orderPath, nil, 200))
	if len(orderData.Lines) != 1 {
		t.Fatalf("order lines = %v", orderData.Lines)
	}
	lineID := orderData.Lines[0].ID
	splitPath := pathID("/sales-order-lines", lineID) + "/work-orders"
	split := func(first, second string) fields {
		return fields{"batches": []fields{
			{"work_order_no": first, "quantity_milli": 6000},
			{"work_order_no": second, "quantity_milli": 4000},
		}}
	}
	api.call(t, "POST", splitPath, split("WO-6", "WO-4"), 409)
	api.call(t, "POST", orderPath+"/confirm", nil, 200)
	api.call(t, "POST", splitPath, split("WO-6", "WO-6"), 409)
	work := readData[[]idResult](t, api.call(t, "POST", splitPath, split("WO-6", "WO-4"), 200))
	if len(work) != 2 {
		t.Fatalf("work orders = %v", work)
	}
	api.call(t, "POST", splitPath, fields{"batches": []fields{{"work_order_no": "WO-EXTRA", "quantity_milli": 1}}}, 409)

	work6, work4 := pathID("/work-orders", work[0].ID), pathID("/work-orders", work[1].ID)
	report := func(path, number string, operation, qty int64) {
		t.Helper()
		api.create(t, path+"/reports", fields{"report_no": number, "operation_id": operation,
			"employee_id": employee, "reported_on": "2026-01-02", "quantity_milli": qty})
	}
	report(work6, "R-6", turning, 6000)
	report(work4, "R-4", turning, 4000)
	api.call(t, "POST", work4+"/reports", fields{"report_no": "R-excess", "operation_id": turning,
		"employee_id": employee, "reported_on": "2026-01-02", "quantity_milli": 1}, 409)
	api.call(t, "DELETE", employeePath, nil, 409)
	api.call(t, "POST", work4+"/complete", fields{"operation_id": turning}, 200)
	job := api.create(t, "/outsource-jobs", fields{"job_no": "OS-1", "work_order_id": work[0].ID, "supplier_id": supplier,
		"operation_id": finishing, "quantity_milli": 6000, "currency": "CNY", "amount_minor": 3001, "sent_on": "2026-01-03"})
	jobPath := pathID("/outsource-jobs", job)
	api.call(t, "POST", work6+"/reports", fields{"report_no": "R-reserved", "operation_id": finishing,
		"employee_id": employee, "reported_on": "2026-01-03", "quantity_milli": 1}, 409)
	api.create(t, jobPath+"/returns", fields{"return_no": "OR-1", "received_on": "2026-01-04", "accepted_milli": 3000})

	statement := func(no string, party int64, direction string, sources ...int64) fields {
		return fields{"statement_no": no, "party_id": party, "currency": "CNY", "direction": direction,
			"period_start": "2026-01-01", "period_end": "2026-01-31", "source_ids": sources}
	}
	payable := statement("AP-1", supplier, "payable", job)
	api.call(t, "POST", "/statements", payable, 400)
	api.call(t, "POST", jobPath+"/returns", fields{"return_no": "OR-2", "received_on": "2026-01-05",
		"accepted_milli": 3000, "rejected_milli": 1000}, 409)
	api.create(t, jobPath+"/returns", fields{"return_no": "OR-2", "received_on": "2026-01-05",
		"accepted_milli": 2000, "rejected_milli": 1000})
	report(work6, "R-rework", finishing, 1000)
	api.call(t, "POST", work6+"/complete", fields{"operation_id": finishing}, 200)

	shipment := func(number string, qty int64) int64 {
		t.Helper()
		return api.create(t, "/shipments", fields{"shipment_no": number, "order_id": order,
			"lines": []fields{{"sales_order_line_id": lineID, "quantity_milli": qty}}})
	}
	ship6, ship4, competing := shipment("SH-6", 6000), shipment("SH-4", 4000), shipment("SH-4B", 4000)
	ship6Confirm := pathID("/shipments", ship6) + "/confirm"
	dispatch := fields{"dispatched_on": "2026-01-06"}
	api.call(t, "POST", "/statements", statement("AR-1", customer, "receivable", ship6), 400)
	api.call(t, "POST", ship6Confirm, dispatch, 200)
	api.call(t, "POST", ship6Confirm, dispatch, 200)
	shippedRest := raceShipments(t, api, dispatch, ship4, competing)
	type shipmentDetail struct {
		Lines []struct {
			Amount int64 `json:"amount_minor"`
		} `json:"lines"`
	}
	for _, shipment := range []struct{ id, amount int64 }{{ship6, 6000}, {shippedRest, 4001}} {
		detail := readData[shipmentDetail](t, api.call(t, "GET", pathID("/shipments", shipment.id), nil, 200))
		if len(detail.Lines) != 1 || detail.Lines[0].Amount != shipment.amount {
			t.Fatalf("shipment %d prices = %+v, want %d", shipment.id, detail.Lines, shipment.amount)
		}
	}
	checkWorkbenchPrinting(t, api, order, work[0].ID, ship6)
	arID := api.create(t, "/statements", statement("AR-1", customer, "receivable", shippedRest, ship6))
	apID := api.create(t, "/statements", payable)
	api.call(t, "POST", "/statements", statement("AR-2", customer, "receivable", ship6), 409)
	checkBalance(t, api, "/statements", arID, 10001, 10001)
	checkBalance(t, api, "/statements", apID, 2500, 2500)

	cash := func(no string, party int64, direction string, amount int64) int64 {
		t.Helper()
		return api.create(t, "/cash-entries", fields{"entry_no": no, "party_id": party, "currency": "CNY",
			"direction": direction, "booked_on": "2026-01-07", "amount_minor": amount, "method": "bank"})
	}
	receipt, payment := cash("RC-1", customer, "receivable", 9000), cash("PY-1", supplier, "payable", 1500)
	allocation := func(key string, statement, entry, amount int64) fields {
		return fields{"request_key": key, "statement_id": statement, "cash_entry_id": entry, "amount_minor": amount}
	}
	api.call(t, "POST", "/allocations", allocation("wrong-account", arID, payment, 1), 409)
	receiptAllocation := allocation("alloc-receipt", arID, receipt, 3000)
	assertConcurrentRetry(t, api, receiptAllocation)
	api.call(t, "POST", "/allocations", allocation("alloc-receipt", arID, receipt, 3001), 409)
	api.create(t, "/allocations", allocation("alloc-payment", apID, payment, 1000))
	api.call(t, "POST", "/allocations", allocation("excess-payment", apID, payment, 501), 409)
	assertSingleSuccess(t,
		parallelPosts(api, "/allocations", allocation("receipt-race-1", arID, receipt, 4000),
			"/allocations", allocation("receipt-race-2", arID, receipt, 4000)))
	checkBalance(t, api, "/statements", arID, 10001, 3001)
	checkBalance(t, api, "/cash-entries", receipt, 9000, 2000)
	checkBalance(t, api, "/statements", apID, 2500, 1500)
	checkBalance(t, api, "/cash-entries", payment, 1500, 500)

	invoiceFor := func(party, st, amount int64, direction, no string) fields {
		return fields{"party_id": party, "statement_id": st, "direction": direction, "invoice_no": no,
			"invoiced_on": "2026-01-08", "currency": "CNY", "gross_amount_minor": amount, "tax_amount_minor": 0}
	}
	api.call(t, "POST", "/invoices", invoiceFor(customer, arID, 10001, "received", "INV-1"), 400)
	issued := api.create(t, "/invoices", invoiceFor(customer, arID, 10001, "issued", "INV-1"))
	received := api.create(t, "/invoices", invoiceFor(supplier, apID, 2500, "received", "INV-2"))
	api.call(t, "POST", pathID("/invoices", received)+"/void", nil, 200)
	invoices := readData[[]fields](t, api.call(t, "GET", fmt.Sprintf("/invoices?after_id=%d", issued), nil, 200))
	if len(invoices) != 1 || invoices[0]["status"] != "void" {
		t.Fatalf("invoice list = %v", invoices)
	}

	orderRequest["order_no"] = "SO-CONCURRENT"
	secondOrder := api.create(t, "/sales-orders", orderRequest)
	secondPath := pathID("/sales-orders", secondOrder)
	api.call(t, "POST", secondPath+"/confirm", nil, 200)
	secondLine := readData[orderDetail](t, api.call(t, "GET", secondPath, nil, 200)).Lines[0].ID
	secondSplit := pathID("/sales-order-lines", secondLine) + "/work-orders"
	assertSingleSuccess(t, parallelPosts(api, secondSplit, split("WO-A6", "WO-A4"),
		secondSplit, split("WO-B6", "WO-B4")))
}

func checkWorkbenchPrinting(t *testing.T, api testAPI, order, work, shipment int64) {
	t.Helper()
	definitions := readData[map[string]printing.Definition](t, api.call(t, "GET", "/print-definitions", nil, 200))
	for _, document := range []struct {
		kind, list, number, quantity string
		id                           int64
	}{
		{"sales-order", "/sales-orders", "SO-1", "10", order},
		{"work-order", "/work-orders", "WO-6", "6", work},
		{"shipment", "/shipments", "SH-6", "6", shipment},
	} {
		list := readData[[]idResult](t, api.call(t, "GET", document.list+"?limit=1", nil, 200))
		if len(list) != 1 || list[0].ID != document.id {
			t.Fatalf("%s first list page = %+v", document.kind, list)
		}
		data := readData[struct {
			Fields map[string]string
			Rows   []map[string]string
		}](t, api.call(t, "GET", pathID("/print-documents/"+document.kind, document.id), nil, 200))
		if data.Fields["number"] != document.number || len(data.Rows) != 1 ||
			data.Rows[0]["quantity"] != document.quantity || data.Rows[0]["product"] != "Example Part" {
			t.Fatalf("%s print data = %+v", document.kind, data)
		}
		if len(definitions[document.kind].Default.Columns) == 0 {
			t.Fatalf("%s has no built-in print template", document.kind)
		}
	}
	api.call(t, "GET", "/print-documents/shipment/999999", nil, 404)
	custom := definitions["sales-order"].Default
	payload := fields{"document_type": "sales-order", "code": "example", "name": "Example template", "configuration": custom}
	first := api.create(t, "/print-templates", payload)
	custom.Title = "Example landscape"
	custom.Orientation = "landscape"
	payload["configuration"] = custom
	second := api.create(t, "/print-templates", payload)
	custom.Columns[0].Field = "unknown_field"
	payload["configuration"] = custom
	api.call(t, "POST", "/print-templates", payload, 400)

	publish := func(id int64) {
		t.Helper()
		api.call(t, "POST", pathID("/print-templates", id)+"/publish", nil, 200)
	}
	publish(first)
	publish(second)
	verify := func(firstActive, secondActive bool) {
		t.Helper()
		versions := readData[[]struct {
			ID      int64
			Version int
			Active  bool
		}](t, api.call(t, "GET", "/print-templates", nil, 200))
		if len(versions) != 2 || versions[0].ID != first || versions[0].Version != 1 || versions[0].Active != firstActive ||
			versions[1].ID != second || versions[1].Version != 2 || versions[1].Active != secondActive {
			t.Fatalf("unexpected print template history %+v", versions)
		}
	}
	verify(false, true)
	publish(first)
	verify(true, false)
}

func checkBalance(t *testing.T, api testAPI, path string, id, amount, remaining int64) {
	t.Helper()
	type balance struct {
		Amount    int64 `json:"amount_minor"`
		Remaining int64 `json:"remaining_minor"`
	}
	got := readData[balance](t, api.call(t, "GET", pathID(path, id), nil, 200))
	if got != (balance{amount, remaining}) {
		t.Fatalf("balance = %+v, want %d total and %d remaining", got, amount, remaining)
	}
}

func parallelPosts(api testAPI, path1 string, first fields, path2 string, second fields) [2]reply {
	start := make(chan struct{})
	result := make(chan reply, 1)
	go func() {
		<-start
		result <- api.send("POST", path1, first)
	}()
	close(start)
	secondReply := api.send("POST", path2, second)
	return [2]reply{<-result, secondReply}
}

func assertSingleSuccess(t *testing.T, results [2]reply) int {
	t.Helper()
	winner := 0
	if results[1].status == 200 {
		winner = 1
	}
	requireReply(t, results[winner], 200)
	requireReply(t, results[1-winner], 409)
	return winner
}

func raceShipments(t *testing.T, api testAPI, dispatch fields, left, right int64) int64 {
	t.Helper()
	ids := [2]int64{left, right}
	results := parallelPosts(api, pathID("/shipments", left)+"/confirm", dispatch,
		pathID("/shipments", right)+"/confirm", dispatch)
	return ids[assertSingleSuccess(t, results)]
}

func assertConcurrentRetry(t *testing.T, api testAPI, allocation fields) {
	t.Helper()
	results := parallelPosts(api, "/allocations", allocation, "/allocations", allocation)
	requireReply(t, results[0], 200)
	requireReply(t, results[1], 200)
	original := readData[idResult](t, results[0])
	if original.ID <= 0 || original != readData[idResult](t, results[1]) {
		t.Fatalf("same key returned different records: %s and %s", results[0].raw, results[1].raw)
	}
}

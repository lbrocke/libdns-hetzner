package hetzner_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/libdns/hetzner"
	"github.com/libdns/libdns"
)

var (
	envToken = ""
	envZone  = ""
	ttl      = 120
)

type testRecordsCleanup = func()

func setupTestRecords(t *testing.T, p *hetzner.Provider) ([]libdns.Record, testRecordsCleanup) {
	testRecords := []hetzner.Record{
		{
			Type:  "TXT",
			Name:  "test1",
			Value: "test1",
			TTL:   ttl,
		}, {
			Type:  "TXT",
			Name:  "test2",
			Value: "test2",
			TTL:   ttl,
		}, {
			Type:  "TXT",
			Name:  "test3",
			Value: "test3",
			TTL:   ttl,
		},
	}

	records, err := p.AppendRecords(context.TODO(), envZone, toLibdnsRecords(testRecords))

	if err != nil {
		t.Fatal(err)
		return nil, func() {}
	}

	return records, func() {
		cleanupRecords(t, p, records)
	}
}

func cleanupRecords(t *testing.T, p *hetzner.Provider, r []libdns.Record) {
	_, err := p.DeleteRecords(context.TODO(), envZone, r)

	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
}

func toLibdnsRecords(records []hetzner.Record) []libdns.Record {
	list := make([]libdns.Record, len(records))

	for i, r := range records {
		list[i], _ = r.Parse(envZone)
	}

	return list
}

func TestMain(m *testing.M) {
	envToken = os.Getenv("LIBDNS_HETZNER_TEST_TOKEN")
	envZone = os.Getenv("LIBDNS_HETZNER_TEST_ZONE")

	if len(envToken) == 0 || len(envZone) == 0 {
		fmt.Println(`Please notice that this test runs agains the public Hetzner DNS Api, so you sould
never run the test with a zone, used in production.
To run this test, you have to specify 'LIBDNS_HETZNER_TEST_TOKEN' and 'LIBDNS_HETZNER_TEST_ZONE'.
Example: "LIBDNS_HETZNER_TEST_TOKEN="123" LIBDNS_HETZNER_TEST_ZONE="my-domain.com" go test ./... -v`)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestProvider_GetRecords(t *testing.T) {
	p := hetzner.New(envToken)
	_, cleanupFunc := setupTestRecords(t, p)
	defer cleanupFunc()
	records, err := p.GetRecords(context.TODO(), envZone)

	if err != nil {
		t.Fatal(err)
	}

	found := 0

	for _, record := range records {
		if record.RR().Name == "test1" || record.RR().Name == "test2" || record.RR().Name == "test3" {
			found++
		}
	}

	if found != 3 {
		t.Fatalf("found %d records, expected 3", found)
	}
}

func TestProvider_AppendRecords(t *testing.T) {
	p := hetzner.New(envToken)
	testCases := []struct {
		records  []hetzner.Record
		expected []hetzner.Record
	}{
		{
			// multiple records
			records: []hetzner.Record{
				{Type: "TXT", Name: "test_1", Value: "test_1", TTL: ttl},
				{Type: "TXT", Name: "test_2", Value: "test_2", TTL: ttl},
				{Type: "TXT", Name: "test_3", Value: "test_3", TTL: ttl},
			},
			expected: []hetzner.Record{
				{Type: "TXT", Name: "test_1", Value: "test_1", TTL: ttl},
				{Type: "TXT", Name: "test_2", Value: "test_2", TTL: ttl},
				{Type: "TXT", Name: "test_3", Value: "test_3", TTL: ttl},
			},
		},
		{
			// relative name
			records: []hetzner.Record{
				{Type: "TXT", Name: "123.test", Value: "123", TTL: ttl},
			},
			expected: []hetzner.Record{
				{Type: "TXT", Name: "123.test", Value: "123", TTL: ttl},
			},
		},
		{
			// (fqdn) sans trailing dot
			records: []hetzner.Record{
				{Type: "TXT", Name: fmt.Sprintf("123.test.%s", strings.TrimSuffix(envZone, ".")), Value: "test", TTL: ttl},
			},
			expected: []hetzner.Record{
				{Type: "TXT", Name: "123.test", Value: "test", TTL: ttl},
			},
		},
		{
			// fqdn with trailing dot
			records: []hetzner.Record{
				{Type: "TXT", Name: fmt.Sprintf("123.test.%s.", strings.TrimSuffix(envZone, ".")), Value: "test", TTL: ttl},
			},
			expected: []hetzner.Record{
				{Type: "TXT", Name: "123.test", Value: "test", TTL: ttl},
			},
		},
	}

	for _, c := range testCases {
		func() {
			result, err := p.AppendRecords(context.TODO(), envZone+".", toLibdnsRecords(c.records))

			if err != nil {
				t.Fatal(err)
			}

			defer cleanupRecords(t, p, result)

			if len(result) != len(c.records) {
				t.Fatalf("len(resilt) != len(c.records) => %d != %d", len(c.records), len(result))
			}

			for k, r := range result {
				rr := r.RR()

				if rr.Type != c.expected[k].Type {
					t.Fatalf("r.Type != c.exptected[%d].Type => %s != %s", k, rr.Type, c.expected[k].Type)
				}

				if rr.Name != c.expected[k].Name {
					t.Fatalf("r.Name != c.exptected[%d].Name => %s != %s", k, rr.Name, c.expected[k].Name)
				}

				if rr.Data != c.expected[k].Value {
					t.Fatalf("r.Value != c.exptected[%d].Value => %s != %s", k, rr.Data, c.expected[k].Value)
				}

				if int(rr.TTL.Seconds()) != c.expected[k].TTL {
					t.Fatalf("r.TTL != c.exptected[%d].TTL => %s != %v", k, rr.TTL, c.expected[k].TTL)
				}
			}
		}()
	}
}

func TestProvider_DeleteRecords(t *testing.T) {
	p := hetzner.New(envToken)
	setupTestRecords(t, p)
	deleted, err := p.DeleteRecords(context.TODO(), envZone, toLibdnsRecords([]hetzner.Record{
		{
			Type:  "TXT",
			Name:  "test2",
			Value: "test2",
			TTL:   ttl,
		},
	}))

	if err != nil {
		t.Fatal(err)
	}

	if len(deleted) != 1 {
		t.Fatalf("len(deleted) != 1 => %d", len(deleted))
	}

	if deleted[0].RR().Name != "test2" {
		t.Fatalf("deleted[0].RR().Name != 'test2' => %s", deleted[0].RR().Name)
	}

	cleanupRecords(t, p, toLibdnsRecords([]hetzner.Record{
		{
			Type: "TXT",
			Name: "test1",
		},
		{
			Type: "TXT",
			Name: "test3",
		},
	}))
}

func TestProvider_SetRecords(t *testing.T) {
	p := hetzner.New(envToken)
	existingRecords, _ := setupTestRecords(t, p)
	newTestRecords := []hetzner.Record{
		{
			Type:  "TXT",
			Name:  "new_test1",
			Value: "new_test1",
			TTL:   ttl,
		},
		{
			Type:  "TXT",
			Name:  "new_test2",
			Value: "new_test2",
			TTL:   ttl,
		},
	}
	allRecords := append(existingRecords, toLibdnsRecords(newTestRecords)...)
	allRecords[0] = &hetzner.Record{
		Type:  allRecords[0].RR().Type,
		Name:  allRecords[0].RR().Name,
		TTL:   int(allRecords[0].RR().TTL.Seconds()),
		Value: "new_value",
	}
	records, err := p.SetRecords(context.TODO(), envZone, allRecords)

	if err != nil {
		t.Fatal(err)
	}

	defer cleanupRecords(t, p, records)

	if len(records) != len(allRecords) {
		t.Fatalf("len(records) != len(allRecords) => %d != %d", len(records), len(allRecords))
	}

	if records[0].RR().Data != "new_value" {
		t.Fatalf(`records[0].Value != "new_value" => %s != "new_value"`, records[0].RR().Data)
	}
}

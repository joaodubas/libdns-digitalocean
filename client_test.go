package digitalocean

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/libdns/libdns"
)

func TestClient_getDNSEntries(t *testing.T) {
	// Mock domain records to return
	mockRecords := []godo.DomainRecord{
		{
			ID:   1,
			Type: "A",
			Name: "test",
			Data: "192.168.1.1",
			TTL:  3600,
		},
		{
			ID:   2,
			Type: "CNAME",
			Name: "www",
			Data: "example.com",
			TTL:  1800,
		},
	}

	// Test successful call
	p := setupTest(setupTestOptions{Records: mockRecords})
	ctx := context.Background()

	records, err := p.getDNSEntries(ctx, "example.com")

	if err != nil {
		t.Errorf("Client.getDNSEntries() error = %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Client.getDNSEntries() returned %d records, want 2", len(records))
	}

	// Verify first record
	if records[0].RR().Type != "A" || records[0].RR().Name != "test" ||
		records[0].RR().Data != "192.168.1.1" || records[0].(DNS).ID != "1" {
		t.Errorf("Client.getDNSEntries()[0] = %v, want A record", records[0])
	}

	// Verify second record
	if records[1].RR().Type != "CNAME" || records[1].RR().Name != "www" ||
		records[1].RR().Data != "example.com" || records[1].(DNS).ID != "2" {
		t.Errorf("Client.getDNSEntries()[1] = %v, want CNAME record", records[1])
	}

	// Test error case
	p = setupTest(setupTestOptions{Error: errors.New("API error")})

	_, err = p.getDNSEntries(ctx, "example.com")
	if err == nil {
		t.Error("Client.getDNSEntries() expected error, got nil")
	}
}

func TestClient_addDNSEntry(t *testing.T) {
	// Test record to add
	testRecord := libdns.RR{
		Type: "A",
		Name: "test",
		Data: "192.168.1.1",
		TTL:  3600 * time.Second,
	}

	// Test successful call
	p := setupTest(setupTestOptions{})
	ctx := context.Background()

	resultRecord, err := p.addDNSEntry(ctx, "example.com", testRecord)

	if err != nil {
		t.Errorf("Client.addDNSEntry() error = %v", err)
	}

	// Verify the returned record
	if resultRecord.RR().Type != testRecord.Type ||
		resultRecord.RR().Name != testRecord.Name ||
		resultRecord.RR().Data != testRecord.Data ||
		resultRecord.(DNS).ID != "12345" {
		t.Errorf("Client.addDNSEntry() record mismatch, got = %v, want Type=%s, Name=%s, Data=%s, ID=12345",
			resultRecord, testRecord.RR().Type, testRecord.RR().Name, testRecord.RR().Data)
	}

	// Test error case
	p = setupTest(setupTestOptions{Error: errors.New("API error")})

	_, err = p.addDNSEntry(ctx, "example.com", testRecord)
	if err == nil {
		t.Error("Client.addDNSEntry() expected error, got nil")
	}
}

func TestClient_removeDNSEntry(t *testing.T) {
	// Test record to delete
	testRecord := DNS{
		ID: "1",
		Record: libdns.RR{
			Type: "A",
			Name: "test",
			Data: "192.168.1.1",
		},
	}

	// Test successful call
	p := setupTest(setupTestOptions{Records: []godo.DomainRecord{{ID: 1}}})
	ctx := context.Background()

	resultRecord, err := p.removeDNSEntry(ctx, "example.com", testRecord)

	if err != nil {
		t.Errorf("Client.removeDNSEntry() error = %v", err)
	}

	// Verify the ID was preserved
	if resultRecord.(DNS).ID != testRecord.ID {
		t.Errorf("Client.removeDNSEntry() ID mismatch, got = %v, want = %v", resultRecord.(DNS).ID, testRecord.ID)
	}

	// Test error case - API error
	p = setupTest(setupTestOptions{Error: errors.New("API error")})

	_, err = p.removeDNSEntry(ctx, "example.com", testRecord)
	if err == nil {
		t.Error("Client.removeDNSEntry() expected error, got nil")
	}

	// Test error case - invalid ID
	p = setupTest(setupTestOptions{})
	invalidIDRecord := DNS{
		ID: "invalid", // Non-numeric ID
		Record: libdns.RR{
			Type: "A",
			Name: "test",
			Data: "192.168.1.1",
		},
	}

	_, err = p.removeDNSEntry(ctx, "example.com", invalidIDRecord)
	if err == nil {
		t.Error("Client.removeDNSEntry() expected error for invalid ID, got nil")
	}
}

func TestClient_updateDNSEntry(t *testing.T) {
	// Test record to update
	testRecord := DNS{
		ID: "1",
		Record: libdns.RR{
			Type: "A",
			Name: "test",
			Data: "192.168.1.2", // Updated IP
			TTL:  7200 * time.Second,
		},
	}

	// Test successful call
	p := setupTest(setupTestOptions{})
	ctx := context.Background()

	resultRecord, err := p.updateDNSEntry(ctx, "example.com", testRecord)

	if err != nil {
		t.Errorf("Client.updateDNSEntry() error = %v", err)
	}

	// Verify the record was preserved
	if resultRecord.(DNS).ID != testRecord.ID ||
		resultRecord.RR().Type != testRecord.RR().Type ||
		resultRecord.RR().Name != testRecord.RR().Name ||
		resultRecord.RR().Data != testRecord.RR().Data {
		t.Errorf("Client.updateDNSEntry() record mismatch, got = %v, want = %v", resultRecord, testRecord)
	}

	// Test error case - API error
	p = setupTest(setupTestOptions{Error: errors.New("API error")})

	_, err = p.updateDNSEntry(ctx, "example.com", testRecord)
	if err == nil {
		t.Error("Client.updateDNSEntry() expected error, got nil")
	}

	// Test error case - invalid ID
	p = setupTest(setupTestOptions{})
	invalidIDRecord := DNS{
		ID: "invalid", // Non-numeric ID
		Record: libdns.RR{
			Type: "A",
			Name: "test",
			Data: "192.168.1.1",
		},
	}

	_, err = p.updateDNSEntry(ctx, "example.com", invalidIDRecord)
	if err == nil {
		t.Error("Client.updateDNSEntry() expected error for invalid ID, got nil")
	}
}

func TestClient_upsertRecord_createRecord(t *testing.T) {
	// Test sucessful creation call
	p := setupTest(setupTestOptions{})
	ctx := context.TODO()
	record := libdns.RR{Type: "A", Name: "test", Data: "192.168.1.1", TTL: 3600 * time.Second}
	resultRecord, err := p.upsertDNSENtry(ctx, "example.com", record)

	if err != nil {
		t.Errorf("Client.upsertDNSENtry() error = %v", err)
	}

	if resultRecord.RR().Type != record.Type ||
		resultRecord.RR().Name != record.Name ||
		resultRecord.RR().Data != record.Data ||
		resultRecord.(DNS).ID != "12345" {
		t.Errorf("Client.upsertDNSENtry() record mismatch, got = %v, want Type=%s, Name=%s, Data=%s, ID=12345",
			resultRecord, record.Type, record.Name, record.Data)
	}

	// Test error in creation call
	p = setupTest(setupTestOptions{MutationError: errors.New("Create API error")})
	ctx = context.TODO()
	_, err = p.upsertDNSENtry(ctx, "example.com", record)

	if err == nil || err.Error() != "Create API error" {
		t.Error("Client.upsertDNSENtry() expected error, got nil")
	}
}

func TestClient_upsertRecord_updateEntry(t *testing.T) {
	mockRecords := []godo.DomainRecord{{ID: 1, Type: "A", Name: "test", Data: "192.168.0.1", TTL: 3600}}
	record := libdns.RR{Type: "A", Name: "test", Data: "192.168.1.1", TTL: 1800 * time.Second}

	// Test successful update call
	p := setupTest(setupTestOptions{Records: mockRecords})
	ctx := context.TODO()
	resultRecord, err := p.upsertDNSENtry(ctx, "example.com", record)

	if err != nil {
		t.Errorf("Client.upsertDNSENtry() error = %v", err)
	}

	if resultRecord.RR().Type != record.Type ||
		resultRecord.RR().Name != record.Name ||
		resultRecord.RR().Data != record.Data ||
		resultRecord.(DNS).ID != "1" {
		t.Errorf("Client.upsertDNSENtry() record mismatch, got = %v, want Type=%s, Name=%s, Data=%s, ID=1",
			resultRecord, record.Type, record.Name, record.Data)
	}

	// Test error in creation call
	p = setupTest(setupTestOptions{Records: mockRecords, MutationError: errors.New("Update API error")})
	ctx = context.TODO()
	_, err = p.upsertDNSENtry(ctx, "example.com", record)

	if err == nil || err.Error() != "Update API error" {
		t.Error("Client.upsertDNSENtry() expected error, got nil")
	}
}

func TestClient_upsertRecord_errorGettingEntries(t *testing.T) {
	mockRecords := []godo.DomainRecord{{ID: 1, Type: "A", Name: "test", Data: "192.168.0.1", TTL: 3600}, {ID: 2, Type: "A", Name: "test", Data: "192.168.0.2", TTL: 3600}}
	record := libdns.RR{Type: "A", Name: "test", Data: "192.168.0.1", TTL: 3600 * time.Second}

	// Test multiple records for same type and name
	p := setupTest(setupTestOptions{Records: mockRecords})
	ctx := context.TODO()
	_, err := p.upsertDNSENtry(ctx, "example.com", record)

	if err == nil {
		t.Error("Client.upsertDNSENtry() expected error, got nil")
	}

	// Test error in getting entries
	p = setupTest(setupTestOptions{QueryError: errors.New("Get API error")})
	ctx = context.TODO()
	_, err = p.upsertDNSENtry(ctx, "example.com", record)

	if err == nil {
		t.Error("Client.upsertDNSENtry() expected error, got nil")
	}
}

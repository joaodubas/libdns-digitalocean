package digitalocean

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/digitalocean/godo"
	"github.com/libdns/libdns"
)

type Client struct {
	client *godo.Client
	mutex  sync.Mutex
}

func (p *Provider) getClient() error {
	if p.client == nil {
		p.client = godo.NewFromToken(p.APIToken)
	}

	return nil
}

func (p *Provider) getDNSEntries(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.getClient()

	opt := &godo.ListOptions{}
	var records []libdns.Record
	for {
		domains, resp, err := p.client.Domains.Records(ctx, zone, opt)
		if err != nil {
			return records, err
		}

		for _, entry := range domains {
			record := fromGodo(entry)
			records = append(records, record)
		}

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return records, err
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}

	return records, nil
}

func (p *Provider) removeDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.getClient()

	// Get record from remote
	record, err := p.getDNSEntry(ctx, zone, record)
	if err != nil {
		return record, err
	}

	// Get ID from dns record
	id, err := idFromRecord(record)
	if err != nil {
		return record, err
	}

	_, err = p.client.Domains.DeleteRecord(ctx, zone, id)
	if err != nil {
		return record, err
	}

	return record, nil
}

func (p *Provider) upsertDNSENtry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.getClient()

	fetchedRecord, err := p.getDNSEntry(ctx, zone, record)

	if err != nil {
		return record, err
	}

	if fetchedRecord.(DNS).ID == "" {
		return p.addDNSEntry(ctx, zone, record)
	}

	record = fromRecord(record, fetchedRecord.(DNS).ID)

	return p.updateDNSEntry(ctx, zone, record)
}

func (p *Provider) getDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	entries, _, err := p.client.Domains.RecordsByTypeAndName(ctx, zone, record.RR().Name, record.RR().Type, &godo.ListOptions{})

	if err != nil {
		return record, err
	} else if len(entries) > 1 {
		return record, fmt.Errorf("found more than one record with name %s and type %s", record.RR().Name, record.RR().Type)
	} else if len(entries) == 0 {
		return fromRecord(record, ""), nil
	}

	record = fromGodo(entries[0])
	return record, nil
}

func (p *Provider) addDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	entry := recordToGoDo(record)

	rec, _, err := p.client.Domains.CreateRecord(ctx, zone, &entry)
	if err != nil {
		return record, err
	}

	return fromRecord(record, strconv.Itoa(rec.ID)), nil
}

func (p *Provider) updateDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	// Get ID from DNS record
	id, err := idFromRecord(record)
	if err != nil {
		return record, err
	}

	entry := recordToGoDo(record)

	_, _, err = p.client.Domains.EditRecord(ctx, zone, id, &entry)
	if err != nil {
		return record, err
	}

	return record, nil
}

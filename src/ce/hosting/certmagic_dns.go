package hosting

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/libdns/libdns"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"go.uber.org/zap"
)

type DNSProvider struct {
	awscli *integrations.AWSClient
	zoneID string
}

// NewDNSProviders returns a new instance of DNSProvider.
// This is only used on Stormkit Cloud.
func NewDNSProvider() *DNSProvider {
	awscli, err := integrations.AWS(integrations.ClientArgs{}, nil)

	if err != nil {
		panic(fmt.Sprintf("cannot create aws cli: %s", err.Error()))
	}

	return &DNSProvider{
		awscli: awscli,
		zoneID: os.Getenv("STORMKT_DEV_ZONE_ID"),
	}
}

func (d *DNSProvider) prepareInput(actionType, zone string, r libdns.RR) *route53.ChangeResourceRecordSetsInput {
	slog.Debug(slog.LogOpts{
		Msg:   "preparing dns record input",
		Level: slog.DL1,
		Payload: []zap.Field{
			zap.String("actionType", actionType),
			zap.String("zone", zone),
			zap.String("name", r.Name),
			zap.String("type", r.Type),
			zap.String("data", r.Data),
			zap.Int64("ttl", int64(r.TTL)),
		},
	})

	return &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					Action: types.ChangeAction(actionType),
					ResourceRecordSet: &types.ResourceRecordSet{
						Name: utils.Ptr(libdns.AbsoluteName(r.Name, zone)),
						ResourceRecords: []types.ResourceRecord{
							{
								Value: utils.Ptr(r.Data),
							},
						},
						TTL:  utils.Ptr(int64(r.TTL)),
						Type: types.RRType(r.Type),
					},
				},
			},
		},
		HostedZoneId: utils.Ptr(d.zoneID),
	}
}

func (d *DNSProvider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	slog.Debug(slog.LogOpts{
		Msg:   "appending record for zone id",
		Level: slog.DL1,
		Payload: []zap.Field{
			zap.String("zone", zone),
		},
	})

	var createdRecords []libdns.Record

	for _, record := range records {
		r := record.RR()

		if r.Type == "TXT" {
			r.Data = strconv.Quote(r.Data)
		}

		input := d.prepareInput("UPSERT", zone, r)
		r.TTL = time.Duration(60) * time.Second

		if _, err := d.awscli.Route53().ChangeResourceRecordSets(ctx, input); err != nil {
			slog.Errorf("error while changing record set=%s, zone=%s, record=%v", err.Error(), zone, record)
			return nil, err
		}

		createdRecords = append(createdRecords, r)

		slog.Debug(slog.LogOpts{
			Msg:   "created dns record",
			Level: slog.DL1,
			Payload: []zap.Field{
				zap.String("name", r.Name),
			},
		})
	}

	return createdRecords, nil
}

// DeleteRecords deletes the records from the zone. If a record does not have an ID,
// it will be looked up. It returns the records that were deleted.
func (d *DNSProvider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var deletedRecords []libdns.Record

	slog.Infof("dns-resolver -- delete operation zone=%s, records=%v", zone, records)

	for _, record := range records {
		input := d.prepareInput("DELETE", zone, record.RR())

		if _, err := d.awscli.Route53().ChangeResourceRecordSets(ctx, input); err != nil {
			slog.Errorf("dns-resolver -- failed deleting %v", err)
			return nil, err
		}

		r := record.RR()
		r.TTL = time.Duration(r.TTL) * time.Second
		deletedRecords = append(deletedRecords, r)
	}

	return deletedRecords, nil
}

package tenant

import (
	"context"

	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/kit/metric"
	"github.com/influxdata/influxdb/kit/prom"
)

type OrgMetrics struct {
	// RED metrics
	rec *metric.REDClient

	orgService influxdb.OrganizationService
}

var _ influxdb.OrganizationService = (*OrgMetrics)(nil)

// NewOrgMetrics returns a metrics service middleware for the Organization Service.
func NewOrgMetrics(reg *prom.Registry, s influxdb.OrganizationService, opts ...MetricsOption) *OrgMetrics {
	o := applyOpts(opts...)
	return &OrgMetrics{
		rec:        metric.New(reg, o.applySuffix("org")),
		orgService: s,
	}
}

func (m *OrgMetrics) FindOrganizationByID(ctx context.Context, id influxdb.ID) (*influxdb.Organization, error) {
	rec := m.rec.Record("find_org_by_id")
	org, err := m.orgService.FindOrganizationByID(ctx, id)
	return org, rec(err)
}

func (m *OrgMetrics) FindOrganization(ctx context.Context, filter influxdb.OrganizationFilter) (*influxdb.Organization, error) {
	rec := m.rec.Record("find_org")
	org, err := m.orgService.FindOrganization(ctx, filter)
	return org, rec(err)
}

func (m *OrgMetrics) FindOrganizations(ctx context.Context, filter influxdb.OrganizationFilter, opt ...influxdb.FindOptions) ([]*influxdb.Organization, int, error) {
	rec := m.rec.Record("find_orgs")
	orgs, n, err := m.orgService.FindOrganizations(ctx, filter, opt...)
	return orgs, n, rec(err)
}

func (m *OrgMetrics) CreateOrganization(ctx context.Context, b *influxdb.Organization) error {
	rec := m.rec.Record("create_org")
	err := m.orgService.CreateOrganization(ctx, b)
	return rec(err)
}

func (m *OrgMetrics) UpdateOrganization(ctx context.Context, id influxdb.ID, upd influxdb.OrganizationUpdate) (*influxdb.Organization, error) {
	rec := m.rec.Record("update_org")
	updatedOrg, err := m.orgService.UpdateOrganization(ctx, id, upd)
	return updatedOrg, rec(err)
}

func (m *OrgMetrics) DeleteOrganization(ctx context.Context, id influxdb.ID) error {
	rec := m.rec.Record("delete_org")
	err := m.orgService.DeleteOrganization(ctx, id)
	return rec(err)
}

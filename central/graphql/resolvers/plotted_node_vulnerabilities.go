package resolvers

import (
	"context"

	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("PlottedNodeVulnerabilities", []string{
			"basicVulnerabilityCounter: VulnerabilityCounter!",
			"vulnerabilities(pagination: Pagination): [NodeVulnerability]!",
		}),
	)
}

// PlottedNodeVulnerabilitiesResolver returns the data required by top risky nodes scatter-plot on vuln mgmt dashboard
type PlottedNodeVulnerabilitiesResolver struct {
	root    *Resolver
	all     []string
	fixable int
}

func newPlottedNodeVulnerabilitiesResolver(ctx context.Context, root *Resolver, args RawQuery) (*PlottedNodeVulnerabilitiesResolver, error) {
	allCveIds, fixableCount, err := getPlottedVulnsIdsAndFixableCount(ctx, root, args)
	if err != nil {
		return nil, err
	}

	return &PlottedNodeVulnerabilitiesResolver{
		root:    root,
		all:     allCveIds,
		fixable: fixableCount,
	}, nil
}

// BasicVulnerabilityCounter returns the vulnCounter for scatter-plot with only total and fixable
func (pvr *PlottedNodeVulnerabilitiesResolver) BasicVulnerabilityCounter(ctx context.Context) (*VulnerabilityCounterResolver, error) {
	return &VulnerabilityCounterResolver{
		all: &VulnerabilityFixableCounterResolver{
			total:   int32(len(pvr.all)),
			fixable: int32(pvr.fixable),
		},
	}, nil
}

// Vulnerabilities returns the node vulnerabilities for top risky nodes scatter-plot
func (pvr *PlottedNodeVulnerabilitiesResolver) Vulnerabilities(ctx context.Context, args PaginatedQuery) ([]NodeVulnerabilityResolver, error) {
	vulnResolvers, err := unwrappedPlottedVulnerabilities(ctx, pvr.root, pvr.all, args)
	if err != nil {
		return nil, err
	}

	ret := make([]NodeVulnerabilityResolver, 0, len(vulnResolvers))
	for _, resolver := range vulnResolvers {
		ret = append(ret, resolver)
	}
	return ret, nil
}

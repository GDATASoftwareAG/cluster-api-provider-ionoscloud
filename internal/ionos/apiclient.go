package ionos

import (
	"context"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
)

var _ IONOSClient = (*APIClient)(nil)

type DatacenterAPI interface {
	CreateDatacenter(ctx context.Context, name, location string) (ionoscloud.Datacenter, *ionoscloud.APIResponse, error)
	GetDatacenter(ctx context.Context, datacenterId string) (ionoscloud.Datacenter, *ionoscloud.APIResponse, error)
	DeleteDatacenter(ctx context.Context, datacenterId string) (*ionoscloud.APIResponse, error)
}

type LanAPI interface {
	CreateLan(ctx context.Context, datacenterId string, public bool) (ionoscloud.LanPost, *ionoscloud.APIResponse, error)
	GetLan(ctx context.Context, datacenterId, lanId string) (ionoscloud.Lan, *ionoscloud.APIResponse, error)
}

type DefaultAPI interface {
	APIInfo(ctx context.Context) (ionoscloud.Info, *ionoscloud.APIResponse, error)
}

type LoadBalancerAPI interface {
	CreateLoadBalancer(ctx context.Context, datacenterId string, loadbalancer ionoscloud.NetworkLoadBalancer) (ionoscloud.NetworkLoadBalancer, *ionoscloud.APIResponse, error)
	GetLoadBalancer(ctx context.Context, datacenterId, loadBalancerId string) (ionoscloud.NetworkLoadBalancer, *ionoscloud.APIResponse, error)
	GetLoadBalancerForwardingRules(ctx context.Context, datacenterId, loadBalancerId string) (ionoscloud.NetworkLoadBalancerForwardingRules, *ionoscloud.APIResponse, error)
	PatchLoadBalancerForwardingRule(ctx context.Context, datacenterId, loadBalancerId, ruleId string, properties ionoscloud.NetworkLoadBalancerForwardingRuleProperties) (ionoscloud.NetworkLoadBalancerForwardingRule, *ionoscloud.APIResponse, error)
}

type ServerAPI interface {
	CreateServer(ctx context.Context, datacenterId string, server ionoscloud.Server) (ionoscloud.Server, *ionoscloud.APIResponse, error)
	GetServer(ctx context.Context, datacenterId, serverId string) (ionoscloud.Server, *ionoscloud.APIResponse, error)
}

//go:generate mockgen -source=$GOFILE -destination=../../mocks/ionoscloud/apiclient.mock.go -package=ionoscloudmocks IONOSClient
type IONOSClient interface {
	DatacenterAPI
	LanAPI
	LoadBalancerAPI
	ServerAPI
	DefaultAPI
}

func NewAPIClient(username, password, token, host string) IONOSClient {
	// TODO: do not create a new client for each reconcile
	cfg := ionoscloud.NewConfiguration(username, password, token, host)

	return &APIClient{
		client: ionoscloud.NewAPIClient(cfg),
	}
}

type APIClient struct {
	client *ionoscloud.APIClient
}

func (c *APIClient) APIInfo(ctx context.Context) (info ionoscloud.Info, response *ionoscloud.APIResponse, err error) {
	return c.client.DefaultApi.ApiInfoGet(ctx).Execute()
}

func (c *APIClient) DeleteDatacenter(ctx context.Context, datacenterId string) (*ionoscloud.APIResponse, error) {
	return c.client.DataCentersApi.DatacentersDelete(ctx, datacenterId).Execute()
}

func (c *APIClient) CreateDatacenter(ctx context.Context, name, location string) (ionoscloud.Datacenter, *ionoscloud.APIResponse, error) {
	datacenter := ionoscloud.Datacenter{
		Properties: &ionoscloud.DatacenterProperties{
			Location: &location,
			Name:     &name,
		},
	}
	datacenterReq := c.client.DataCentersApi.DatacentersPost(ctx)
	return datacenterReq.Datacenter(datacenter).Execute()
}

func (c *APIClient) CreateLan(ctx context.Context, datacenterId string, public bool) (ionoscloud.LanPost, *ionoscloud.APIResponse, error) {
	lan := ionoscloud.LanPost{
		Properties: &ionoscloud.LanPropertiesPost{
			Public: ionoscloud.ToPtr(public),
		},
	}
	lanReq := c.client.LANsApi.DatacentersLansPost(ctx, datacenterId)
	return lanReq.Lan(lan).Execute()
}

func (c *APIClient) GetLan(ctx context.Context, datacenterId, lanId string) (ionoscloud.Lan, *ionoscloud.APIResponse, error) {
	lanReq := c.client.LANsApi.DatacentersLansFindById(ctx, datacenterId, lanId)
	return lanReq.Execute()
}

func (c *APIClient) GetDatacenter(ctx context.Context, datacenterId string) (ionoscloud.Datacenter, *ionoscloud.APIResponse, error) {
	return c.client.DataCentersApi.DatacentersFindById(ctx, datacenterId).Execute()
}

func (c *APIClient) CreateLoadBalancer(ctx context.Context, datacenterId string, loadBalancer ionoscloud.NetworkLoadBalancer) (ionoscloud.NetworkLoadBalancer, *ionoscloud.APIResponse, error) {
	loadBalancerReq := c.client.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersPost(ctx, datacenterId)
	return loadBalancerReq.NetworkLoadBalancer(loadBalancer).Execute()
}

func (c *APIClient) GetLoadBalancer(ctx context.Context, datacenterId, loadBalancerId string) (ionoscloud.NetworkLoadBalancer, *ionoscloud.APIResponse, error) {
	loadBalancerReq := c.client.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFindByNetworkLoadBalancerId(ctx, datacenterId, loadBalancerId)
	return loadBalancerReq.Execute()
}

func (c *APIClient) GetLoadBalancerForwardingRules(ctx context.Context, datacenterId, loadBalancerId string) (ionoscloud.NetworkLoadBalancerForwardingRules, *ionoscloud.APIResponse, error) {
	loadBalancerReq := c.client.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesGet(ctx, datacenterId, loadBalancerId)
	return loadBalancerReq.Depth(2).Execute()
}

func (c *APIClient) PatchLoadBalancerForwardingRule(ctx context.Context, datacenterId, loadBalancerId, ruleId string, properties ionoscloud.NetworkLoadBalancerForwardingRuleProperties) (ionoscloud.NetworkLoadBalancerForwardingRule, *ionoscloud.APIResponse, error) {
	updateReq := c.client.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesPatch(ctx, datacenterId, loadBalancerId, ruleId)
	return updateReq.NetworkLoadBalancerForwardingRuleProperties(properties).Execute()
}

func (c *APIClient) CreateServer(ctx context.Context, datacenterId string, server ionoscloud.Server) (ionoscloud.Server, *ionoscloud.APIResponse, error) {
	serverReq := c.client.ServersApi.DatacentersServersPost(ctx, datacenterId)
	return serverReq.Server(server).Execute()
}

func (c *APIClient) GetServer(ctx context.Context, datacenterId, serverId string) (ionoscloud.Server, *ionoscloud.APIResponse, error) {
	serverReq := c.client.ServersApi.DatacentersServersFindById(ctx, datacenterId, serverId)
	return serverReq.Depth(2).Execute()
}

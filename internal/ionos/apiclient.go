package ionos

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
)

type key int

const (
	DepthKey key = iota
)

var _ Client = (*APIClient)(nil)

var (
	clients = map[string]*ionoscloud.APIClient{}
	mutex   = sync.Mutex{}
)

type DatacenterAPI interface {
	CreateDatacenter(ctx context.Context, name string, location v1alpha1.Location) (ionoscloud.Datacenter, *ionoscloud.APIResponse, error)
	GetDatacenter(ctx context.Context, datacenterId string) (ionoscloud.Datacenter, *ionoscloud.APIResponse, error)
	DeleteDatacenter(ctx context.Context, datacenterId string) (*ionoscloud.APIResponse, error)
}

type IPBlockAPI interface {
	GetIPBlock(ctx context.Context, id string) (ionoscloud.IpBlock, *ionoscloud.APIResponse, error)
}

type LanAPI interface {
	CreateLan(ctx context.Context, datacenterId string, public bool) (ionoscloud.LanPost, *ionoscloud.APIResponse, error)
	GetLan(ctx context.Context, datacenterId, lanId string) (ionoscloud.Lan, *ionoscloud.APIResponse, error)
	PatchLanWithIPFailover(ctx context.Context, datacenterId, lanId string, ipFailover []ionoscloud.IPFailover) error
}

type DefaultAPI interface {
	APIInfo(ctx context.Context) (ionoscloud.Info, *ionoscloud.APIResponse, error)
}

type VolumeAPI interface {
	DeleteVolume(ctx context.Context, datacenterId, volumeId string) (*ionoscloud.APIResponse, error)
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
	DeleteServer(ctx context.Context, datacenterId, serverId string) (*ionoscloud.APIResponse, error)
	PatchServerNicsWithIPs(ctx context.Context, datacenterId, serverId, nicUuid string, ips []string) error
}

type Client interface {
	DatacenterAPI
	LanAPI
	LoadBalancerAPI
	ServerAPI
	DefaultAPI
	VolumeAPI
	IPBlockAPI
}

func NewAPIClient(username, password, token, host string) Client {
	mutex.Lock()
	defer func() {
		mutex.Unlock()
	}()

	key := fmt.Sprintf("%s|%s|%s|%s", username, password, token, host)
	if _, ok := clients[key]; !ok {
		cfg := ionoscloud.NewConfiguration(username, password, token, host)
		clients[key] = ionoscloud.NewAPIClient(cfg)
	}

	return &APIClient{
		client: clients[key],
	}
}

type APIClient struct {
	client *ionoscloud.APIClient
}

func (c *APIClient) PatchLanWithIPFailover(ctx context.Context, datacenterId, lanId string, ipFailover []ionoscloud.IPFailover) error {
	_, _, err := c.client.LANsApi.DatacentersLansPatch(ctx, datacenterId, lanId).Lan(ionoscloud.LanProperties{
		IpFailover: &ipFailover,
	}).Execute()
	return err
}

func (c *APIClient) PatchServerNicsWithIPs(ctx context.Context, datacenterId, serverId, nicUuid string, ips []string) error {
	serverId = strings.TrimPrefix(serverId, "ionos://")
	_, _, err := c.client.NetworkInterfacesApi.DatacentersServersNicsPatch(ctx, datacenterId, serverId, nicUuid).Nic(ionoscloud.NicProperties{
		Ips: &ips,
	}).Execute()
	return err
}

func (c *APIClient) GetIPBlock(ctx context.Context, id string) (ionoscloud.IpBlock, *ionoscloud.APIResponse, error) {
	return c.client.IPBlocksApi.IpblocksFindById(ctx, id).Execute()
}

func (c *APIClient) DeleteVolume(ctx context.Context, datacenterId, volumeId string) (*ionoscloud.APIResponse, error) {
	return c.client.VolumesApi.DatacentersVolumesDelete(ctx, datacenterId, volumeId).Execute()
}

func (c *APIClient) DeleteServer(ctx context.Context, datacenterId, serverId string) (*ionoscloud.APIResponse, error) {
	serverId = strings.TrimPrefix(serverId, "ionos://")
	return c.client.ServersApi.DatacentersServersDelete(ctx, datacenterId, serverId).Execute()
}

func (c *APIClient) APIInfo(ctx context.Context) (info ionoscloud.Info, response *ionoscloud.APIResponse, err error) {
	return c.client.DefaultApi.ApiInfoGet(ctx).Execute()
}

func (c *APIClient) DeleteDatacenter(ctx context.Context, datacenterId string) (*ionoscloud.APIResponse, error) {
	return c.client.DataCentersApi.DatacentersDelete(ctx, datacenterId).Execute()
}

func (c *APIClient) CreateDatacenter(ctx context.Context, name string, location v1alpha1.Location) (ionoscloud.Datacenter, *ionoscloud.APIResponse, error) {
	datacenter := ionoscloud.Datacenter{
		Properties: &ionoscloud.DatacenterProperties{
			Location: ionoscloud.ToPtr(location.String()),
			Name:     &name,
		},
	}
	return c.client.DataCentersApi.
		DatacentersPost(ctx).
		Datacenter(datacenter).
		Execute()
}

func (c *APIClient) CreateLan(ctx context.Context, datacenterId string, public bool) (ionoscloud.LanPost, *ionoscloud.APIResponse, error) {
	lan := ionoscloud.LanPost{
		Properties: &ionoscloud.LanPropertiesPost{
			Public: ionoscloud.ToPtr(public),
		},
	}
	return c.client.LANsApi.
		DatacentersLansPost(ctx, datacenterId).
		Lan(lan).
		Execute()
}

func (c *APIClient) GetLan(ctx context.Context, datacenterId, lanId string) (ionoscloud.Lan, *ionoscloud.APIResponse, error) {
	u, ok := ctx.Value(DepthKey).(int32)
	if !ok {
		u = 1
	}
	return c.client.LANsApi.DatacentersLansFindById(ctx, datacenterId, lanId).Depth(u).Execute()
}

func (c *APIClient) GetDatacenter(ctx context.Context, datacenterId string) (ionoscloud.Datacenter, *ionoscloud.APIResponse, error) {
	return c.client.DataCentersApi.DatacentersFindById(ctx, datacenterId).Execute()
}

func (c *APIClient) CreateLoadBalancer(ctx context.Context, datacenterId string, loadBalancer ionoscloud.NetworkLoadBalancer) (ionoscloud.NetworkLoadBalancer, *ionoscloud.APIResponse, error) {
	return c.client.NetworkLoadBalancersApi.
		DatacentersNetworkloadbalancersPost(ctx, datacenterId).
		NetworkLoadBalancer(loadBalancer).
		Execute()
}

func (c *APIClient) GetLoadBalancer(ctx context.Context, datacenterId, loadBalancerId string) (ionoscloud.NetworkLoadBalancer, *ionoscloud.APIResponse, error) {
	return c.client.NetworkLoadBalancersApi.
		DatacentersNetworkloadbalancersFindByNetworkLoadBalancerId(ctx, datacenterId, loadBalancerId).
		Execute()
}

func (c *APIClient) GetLoadBalancerForwardingRules(ctx context.Context, datacenterId, loadBalancerId string) (ionoscloud.NetworkLoadBalancerForwardingRules, *ionoscloud.APIResponse, error) {
	return c.client.NetworkLoadBalancersApi.
		DatacentersNetworkloadbalancersForwardingrulesGet(ctx, datacenterId, loadBalancerId).
		Depth(2).
		Execute()
}

func (c *APIClient) PatchLoadBalancerForwardingRule(ctx context.Context, datacenterId, loadBalancerId, ruleId string, properties ionoscloud.NetworkLoadBalancerForwardingRuleProperties) (ionoscloud.NetworkLoadBalancerForwardingRule, *ionoscloud.APIResponse, error) {
	return c.client.NetworkLoadBalancersApi.
		DatacentersNetworkloadbalancersForwardingrulesPatch(ctx, datacenterId, loadBalancerId, ruleId).
		NetworkLoadBalancerForwardingRuleProperties(properties).
		Execute()
}

func (c *APIClient) CreateServer(ctx context.Context, datacenterId string, server ionoscloud.Server) (ionoscloud.Server, *ionoscloud.APIResponse, error) {
	serverReq := c.client.ServersApi.DatacentersServersPost(ctx, datacenterId)
	server, response, err := serverReq.Server(server).Execute()
	if server.Id != nil {
		server.Id = ionoscloud.ToPtr(fmt.Sprintf("ionos://%s", *server.Id))
	}
	return server, response, err
}

func (c *APIClient) GetServer(ctx context.Context, datacenterId, serverId string) (ionoscloud.Server, *ionoscloud.APIResponse, error) {
	serverId = strings.TrimPrefix(serverId, "ionos://")
	serverReq := c.client.ServersApi.DatacentersServersFindById(ctx, datacenterId, serverId)
	server, resp, err := serverReq.Depth(2).Execute()
	if server.Id != nil {
		server.Id = ionoscloud.ToPtr(fmt.Sprintf("ionos://%s", *server.Id))
	}
	return server, resp, err
}

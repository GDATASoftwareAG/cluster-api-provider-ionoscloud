package testing

import (
	"context"
	"fmt"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/ionos"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/uuid"
	"net/http"
	"strings"
)

var _ ionos.Client = (*FakeClient)(nil)

func NewFakeClient() *FakeClient {
	return &FakeClient{
		DataCenters:         map[string]*ionoscloud.Datacenter{},
		CredentialsAreValid: true,
	}
}

type FakeClient struct {
	DataCenters         map[string]*ionoscloud.Datacenter
	CredentialsAreValid bool
}

func (f FakeClient) PatchLanWithIPFailover(_ context.Context, _, _ string, _ []ionoscloud.IPFailover) error {
	//TODO implement me
	panic("implement me")
}

func (f FakeClient) PatchServerNicsWithIPs(_ context.Context, _, _, _ string, _ []string) error {
	//TODO implement me
	panic("implement me")
}

func (f FakeClient) GetIPBlock(_ context.Context, _ string) (ionoscloud.IpBlock, *ionoscloud.APIResponse, error) {
	return ionoscloud.IpBlock{}, nil, nil
}

func (f FakeClient) DeleteServer(_ context.Context, datacenterId, serverId string) (*ionoscloud.APIResponse, error) {
	serverId = strings.TrimPrefix(serverId, "ionos://")
	var items []ionoscloud.Server

	for _, server := range *f.DataCenters[datacenterId].Entities.Servers.Items {
		if *server.Id != serverId {
			items = append(items, server)
		}
	}

	f.DataCenters[datacenterId].Entities.Servers.Items = &items
	return &ionoscloud.APIResponse{Response: &http.Response{
		StatusCode: http.StatusOK,
	}}, nil
}

func (f FakeClient) DeleteVolume(_ context.Context, _, _ string) (*ionoscloud.APIResponse, error) {
	return &ionoscloud.APIResponse{Response: &http.Response{
		StatusCode: http.StatusOK,
	}}, nil
}

func (f FakeClient) APIInfo(_ context.Context) (info ionoscloud.Info, response *ionoscloud.APIResponse, err error) {
	if f.CredentialsAreValid {
		return *ionoscloud.NewInfo(), ionoscloud.NewAPIResponse(nil), nil
	}

	return *ionoscloud.NewInfo(), ionoscloud.NewAPIResponseWithError("invalid credentials"), errors.New("invalid credentials")
}

func (f FakeClient) CreateLan(_ context.Context, datacenterId string, public bool) (ionoscloud.LanPost, *ionoscloud.APIResponse, error) {
	id := fmt.Sprint(len(*f.DataCenters[datacenterId].Entities.Lans.Items))
	lan := ionoscloud.Lan{
		Id: ionoscloud.ToPtr(id),
		Metadata: &ionoscloud.DatacenterElementMetadata{
			State: ionoscloud.ToPtr("AVAILABLE"),
		},
		Properties: &ionoscloud.LanProperties{
			Public: ionoscloud.ToPtr(public),
		},
	}

	items := append(*f.DataCenters[datacenterId].Entities.Lans.Items, lan)
	f.DataCenters[datacenterId].Entities.Lans.Items = &items
	lan.Id = ionoscloud.ToPtr(id)

	return ionoscloud.LanPost{
			Id: ionoscloud.ToPtr(id),
			Metadata: &ionoscloud.DatacenterElementMetadata{
				State: ionoscloud.ToPtr("AVAILABLE"),
			}},
		&ionoscloud.APIResponse{Response: &http.Response{
			StatusCode: http.StatusOK,
		}},
		nil
}

func (f FakeClient) GetLan(_ context.Context, datacenterId, lanId string) (ionoscloud.Lan, *ionoscloud.APIResponse, error) {
	lans := *f.DataCenters[datacenterId].Entities.Lans.Items

	for _, lan := range lans {
		if *lan.Id == lanId {
			return lan, &ionoscloud.APIResponse{Response: &http.Response{
					StatusCode: http.StatusOK,
				}},
				nil
		}
	}

	return ionoscloud.Lan{}, &ionoscloud.APIResponse{Response: &http.Response{
			StatusCode: http.StatusOK,
		}},
		nil
}

func (f FakeClient) CreateLoadBalancer(_ context.Context, datacenterId string, loadbalancer ionoscloud.NetworkLoadBalancer) (ionoscloud.NetworkLoadBalancer, *ionoscloud.APIResponse, error) {
	if _, ok := f.DataCenters[datacenterId]; !ok {
		return loadbalancer, &ionoscloud.APIResponse{Response: &http.Response{
			StatusCode: http.StatusBadRequest,
		}}, nil
	}

	uid := string(uuid.NewUUID())
	loadbalancer.Id = ionoscloud.ToPtr(uid)
	loadbalancer.Metadata = &ionoscloud.DatacenterElementMetadata{
		State: ionoscloud.ToPtr("AVAILABLE"),
	}

	rules := []ionoscloud.NetworkLoadBalancerForwardingRule{}
	for i, rule := range *loadbalancer.Entities.Forwardingrules.Items {
		rule.Id = ionoscloud.ToPtr(fmt.Sprint(i))
		rules = append(rules, rule)
	}

	loadbalancer.Entities.Forwardingrules.Items = &rules

	items := append(*f.DataCenters[datacenterId].Entities.Networkloadbalancers.Items, loadbalancer)
	f.DataCenters[datacenterId].Entities.Networkloadbalancers.Items = &items

	return loadbalancer,
		&ionoscloud.APIResponse{Response: &http.Response{
			StatusCode: http.StatusOK,
		}},
		nil
}

func (f FakeClient) GetLoadBalancer(_ context.Context, datacenterId, loadBalancerId string) (ionoscloud.NetworkLoadBalancer, *ionoscloud.APIResponse, error) {
	loadbalancers := *f.DataCenters[datacenterId].Entities.Networkloadbalancers.Items

	for _, loadbalancer := range loadbalancers {
		if *loadbalancer.Id == loadBalancerId {
			return loadbalancer, &ionoscloud.APIResponse{Response: &http.Response{
					StatusCode: http.StatusOK,
				}},
				nil
		}
	}

	return ionoscloud.NetworkLoadBalancer{}, &ionoscloud.APIResponse{Response: &http.Response{
			StatusCode: http.StatusOK,
		}},
		nil
}

func (f FakeClient) GetLoadBalancerForwardingRules(_ context.Context, datacenterId, loadBalancerId string) (ionoscloud.NetworkLoadBalancerForwardingRules, *ionoscloud.APIResponse, error) {
	loadbalancers := *f.DataCenters[datacenterId].Entities.Networkloadbalancers.Items

	for _, loadbalancer := range loadbalancers {
		if *loadbalancer.Id == loadBalancerId {
			return *loadbalancer.Entities.Forwardingrules, &ionoscloud.APIResponse{Response: &http.Response{
					StatusCode: http.StatusOK,
				}},
				nil
		}
	}

	return ionoscloud.NetworkLoadBalancerForwardingRules{}, &ionoscloud.APIResponse{Response: &http.Response{
			StatusCode: http.StatusOK,
		}},
		nil
}

func (f FakeClient) PatchLoadBalancerForwardingRule(_ context.Context, datacenterId, loadBalancerId, ruleId string, properties ionoscloud.NetworkLoadBalancerForwardingRuleProperties) (ionoscloud.NetworkLoadBalancerForwardingRule, *ionoscloud.APIResponse, error) {
	loadbalancers := *f.DataCenters[datacenterId].Entities.Networkloadbalancers.Items

	for _, loadbalancer := range loadbalancers {
		if *loadbalancer.Id == loadBalancerId {
			for _, rule := range *loadbalancer.Entities.Forwardingrules.Items {
				if *rule.Id == ruleId {
					rule.Properties = &properties
				} else {
					return rule, &ionoscloud.APIResponse{Response: &http.Response{
							StatusCode: http.StatusOK,
						}},
						nil
				}
			}
		}
	}

	return ionoscloud.NetworkLoadBalancerForwardingRule{}, &ionoscloud.APIResponse{Response: &http.Response{
			StatusCode: http.StatusOK,
		}},
		nil
}

func (f FakeClient) CreateServer(_ context.Context, datacenterId string, server ionoscloud.Server) (ionoscloud.Server, *ionoscloud.APIResponse, error) {
	if _, ok := f.DataCenters[datacenterId]; !ok {
		return server, &ionoscloud.APIResponse{Response: &http.Response{
			StatusCode: http.StatusBadRequest,
		}}, errors.New("BadRequest")
	}

	uid := string(uuid.NewUUID())
	server.Id = ionoscloud.ToPtr(uid)
	server.Metadata = &ionoscloud.DatacenterElementMetadata{
		State: ionoscloud.ToPtr("AVAILABLE"),
	}
	for _, nic := range *server.Entities.Nics.Items {
		ips := []string{"123.123.123.123"}
		nic.Properties.Ips = &ips
	}

	items := append(*f.DataCenters[datacenterId].Entities.Servers.Items, server)
	f.DataCenters[datacenterId].Entities.Servers.Items = &items

	return server,
		&ionoscloud.APIResponse{Response: &http.Response{
			StatusCode: http.StatusOK,
		}},
		nil
}

func (f FakeClient) GetServer(_ context.Context, datacenterId, serverId string) (ionoscloud.Server, *ionoscloud.APIResponse, error) {
	serverId = strings.TrimPrefix(serverId, "ionos://")
	servers := *f.DataCenters[datacenterId].Entities.Servers.Items

	for _, server := range servers {
		if *server.Id == serverId {
			return server, &ionoscloud.APIResponse{Response: &http.Response{
					StatusCode: http.StatusOK,
				}},
				nil
		}
	}

	return ionoscloud.Server{}, &ionoscloud.APIResponse{Response: &http.Response{
			StatusCode: http.StatusNotFound,
		}},
		errors.New("not found")
}

func (f FakeClient) CreateDatacenter(_ context.Context, name string, location v1alpha1.Location) (ionoscloud.Datacenter, *ionoscloud.APIResponse, error) {
	uid := string(uuid.NewUUID())
	dc := ionoscloud.Datacenter{
		Id: ionoscloud.ToPtr(uid),
		Metadata: &ionoscloud.DatacenterElementMetadata{
			State: ionoscloud.ToPtr("AVAILABLE"),
		},
		Entities: &ionoscloud.DataCenterEntities{
			Servers: &ionoscloud.Servers{
				Items: &[]ionoscloud.Server{},
			},
			Lans: &ionoscloud.Lans{
				Items: &[]ionoscloud.Lan{},
			},
			Networkloadbalancers: &ionoscloud.NetworkLoadBalancers{
				Items: &[]ionoscloud.NetworkLoadBalancer{},
			},
		},
		Properties: &ionoscloud.DatacenterProperties{
			Location: ionoscloud.ToPtr(location.String()),
			Name:     ionoscloud.ToPtr(name),
		},
	}
	f.DataCenters[uid] = &dc

	return dc,
		&ionoscloud.APIResponse{Response: &http.Response{
			StatusCode: http.StatusOK,
		}},
		nil
}

func (f FakeClient) GetDatacenter(_ context.Context, datacenterId string) (ionoscloud.Datacenter, *ionoscloud.APIResponse, error) {
	dc, ok := f.DataCenters[datacenterId]

	if !ok {
		return ionoscloud.Datacenter{},
			&ionoscloud.APIResponse{
				Response: &http.Response{
					StatusCode: http.StatusNotFound,
				},
			},
			errors.New("Not Found")
	}

	return *dc,
		&ionoscloud.APIResponse{
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		},
		nil
}

func (f FakeClient) DeleteDatacenter(_ context.Context, datacenterId string) (*ionoscloud.APIResponse, error) {
	f.DataCenters[datacenterId] = nil

	return &ionoscloud.APIResponse{
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		},
		nil
}

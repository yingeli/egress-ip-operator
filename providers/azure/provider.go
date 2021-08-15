package azure

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/yingeli/egress-ip-operator/providers/azure/compute"
	"github.com/yingeli/egress-ip-operator/providers/azure/imds"
	"github.com/yingeli/egress-ip-operator/providers/azure/internal/config"
	"github.com/yingeli/egress-ip-operator/providers/azure/network"
	//ctrl "sigs.k8s.io/controller-runtime"
)

const (
	ipconfigPrefix = "ipconfig-egress-ip"
)

var (
//log = ctrl.Log.WithName("setup")
)

type Provider struct {
	vm string
}

func NewProvider() Provider {
	return Provider{}
}

func (p *Provider) Associate(ctx context.Context, publicIPAddr string, localIPAddr string) (sourceIPAddr string, err error) {
	if !p.initialized() {
		if err := p.initialize(); err != nil {
			return sourceIPAddr, err
		}
	}
	return p.associate(ctx, publicIPAddr, localIPAddr)
}

func (p *Provider) Dissociate(ctx context.Context, sourceIPAddr string) error {
	if !p.initialized() {
		if err := p.initialize(); err != nil {
			return err
		}
	}
	return p.dissociate(ctx, sourceIPAddr)
}

func (p *Provider) initialized() bool {
	return p.vm != ""
}

func (p *Provider) initialize() error {
	err := config.ParseEnvironment()
	if err != nil {
		return fmt.Errorf("config.ParseEnvironment error: %v", err)
	}

	metadata, err := imds.GetMetadata()
	if err != nil {
		return fmt.Errorf("imds.GetMetadata error: %v", err)
	}
	compute := metadata.Compute

	config.SetGroup(compute.AzEnvironment, compute.SubscriptionId, compute.ResourceGroupName)

	p.vm = compute.Name

	return nil
}

func (p *Provider) associate(ctx context.Context, publicIPAddr string, localIPAddr string) (sourceIPAddr string, err error) {
	pip, found, err := network.LookupPublicIP(ctx, publicIPAddr)
	if err != nil {
		return sourceIPAddr, fmt.Errorf("LookupPublicIP error: %v", err)
	}
	if !found {
		return sourceIPAddr, fmt.Errorf("LookupPublicIP cannot find public ip %s", publicIPAddr)
	}

	if pip.IPConfiguration != nil {
		r, err := network.ParseIPConfigurationID(*pip.IPConfiguration.ID)
		if err != nil {
			return sourceIPAddr, fmt.Errorf("ParseIPConfigurationID error: %v", err)
		}
		nic, err := network.GetNic(ctx, r.NicName)
		if err != nil {
			return sourceIPAddr, fmt.Errorf("GetNic error: %v", err)
		}
		err = network.DissociateNicPublicIP(ctx, nic, *pip.IPAddress, ipconfigPrefix)
		if err != nil {
			return sourceIPAddr, fmt.Errorf("DissociateNicIPConfigurationWithPublicIP error: %v", err)
		}
	}

	vm, err := compute.GetVM(ctx, p.vm)
	if err != nil {
		return sourceIPAddr, fmt.Errorf("GetVM error: %v", err)
	}

	for _, ni := range *vm.NetworkProfile.NetworkInterfaces {
		resource, err := azure.ParseResourceID(*ni.ID)
		if err != nil {
			return sourceIPAddr, fmt.Errorf("ParseResourceID error: %v", err)
		}

		nic, err := network.GetNic(ctx, resource.ResourceName)
		if err != nil {
			return sourceIPAddr, fmt.Errorf("GetNic error: %v", err)
		}

		if nic.Primary == nil || *nic.Primary {
			return network.AssociateNicWithPublicIP(ctx, nic, pip, localIPAddr, getIPConfigurationName(*pip.Name))
		}
	}
	return sourceIPAddr, fmt.Errorf("cannot find primary nic on VM %s", p.vm)
}

func (p *Provider) dissociate(ctx context.Context, privateIPAddr string) error {
	vm, err := compute.GetVM(ctx, p.vm)
	if err != nil {
		return fmt.Errorf("GetVM error: %v", err)
	}

	for _, ni := range *vm.NetworkProfile.NetworkInterfaces {
		resource, err := azure.ParseResourceID(*ni.ID)
		if err != nil {
			return fmt.Errorf("ParseResourceID error: %v", err)
		}

		nic, err := network.GetNic(ctx, resource.ResourceName)
		if err != nil {
			return fmt.Errorf("GetNic error: %v", err)
		}

		if nic.Primary == nil || *nic.Primary {
			return network.DissociateNicPublicIPWithPrivateIP(ctx, nic, privateIPAddr, ipconfigPrefix)
		}
	}
	return fmt.Errorf("cannot find primary nic on VM %s", p.vm)
}

/*
func (p *Provider) dissociate(ctx context.Context, publicIPAddr string, privateIPAddr string) error {
	pip, found, err := network.LookupPublicIP(ctx, publicIPAddr)
	if err != nil {
		return fmt.Errorf("LookupPublicIP error: %v", err)
	}
	if !found {
		return nil
	}

	ipconfig, err := network.GetIPConfiguration(ctx, *pip.IPConfiguration.ID)
	if err != nil {
		return fmt.Errorf("GetIPConfiguration error: %v", err)
	}

	if *ipconfig.PrivateIPAddress != privateIPAddr {
		return nil
	}

	delete := isCreated(*ipconfig.Name)
	err = network.DissociateNicIPConfigurationWithPublicIP(ctx, *ipconfig.ID, delete)
	if err != nil {
		return fmt.Errorf("DissociateNicIPConfigurationWithPublicIP error: %v", err)
	}
	return nil
}
*/

func getIPConfigurationName(pipName string) string {
	return ipconfigPrefix + "_" + pipName
}

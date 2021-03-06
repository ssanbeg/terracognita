package azurerm

import (
	"context"

	"github.com/pkg/errors"

	"github.com/cycloidio/terracognita/filter"
	"github.com/cycloidio/terracognita/provider"
)

// ResourceType is the type used to define all the Resources
// from the Provider
type ResourceType int

//go:generate enumer -type ResourceType -addprefix azurerm_ -transform snake -linecomment
const (
	ResourceGroup ResourceType = iota
	Subnet
	VirtualDesktopHostPool
	VirtualDesktopApplicationGroup
	LogicAppTriggerCustom
	LogicAppActionCustom
	LogicAppWorkflow
	NetworkInterface
	NetworkSecurityGroup
	VirtualMachine
	VirtualMachineExtension
	VirtualMachineScaleSet
	VirtualNetwork
)

type rtFn func(ctx context.Context, a *azurerm, resourceType string, filters *filter.Filter) ([]provider.Resource, error)

var (
	resources = map[ResourceType]rtFn{
		ResourceGroup:                  resourceGroup,
		VirtualMachine:                 virtualMachines,
		VirtualMachineExtension:        virtualMachineExtensions,
		VirtualNetwork:                 cacheVirtualNetworks,
		Subnet:                         subnets,
		LogicAppTriggerCustom:          logicAppTriggerCustoms,
		LogicAppActionCustom:           logicAppActionCustoms,
		LogicAppWorkflow:               logicAppWorkflows,
		NetworkInterface:               networkInterfaces,
		NetworkSecurityGroup:           networkSecurityGroups,
		VirtualMachineScaleSet:         virtualMachineScaleSets,
		VirtualDesktopApplicationGroup: virtualApplicationGroups,
		VirtualDesktopHostPool:         virtualDesktopHostPools,
	}
)

func resourceGroup(ctx context.Context, a *azurerm, resourceType string, filters *filter.Filter) ([]provider.Resource, error) {
	resourceGroup := a.azurer.GetResourceGroup()
	r := provider.NewResource(*resourceGroup.ID, resourceType, a)
	resources := []provider.Resource{r}
	return resources, nil
}

func virtualMachines(ctx context.Context, a *azurerm, resourceType string, filters *filter.Filter) ([]provider.Resource, error) {
	virtualMachines, err := a.azurer.ListVirtualMachines(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list virtual machines from reader")
	}
	resources := make([]provider.Resource, 0, len(virtualMachines))
	for _, virtualMachine := range virtualMachines {
		r := provider.NewResource(*virtualMachine.ID, resourceType, a)
		if err := r.Data().Set("name", *virtualMachine.Name); err != nil {
			return nil, errors.Wrapf(err, "unable to set name data on the provider.Resource for the virtual machine '%s'", *virtualMachine.Name)
		}
		resources = append(resources, r)
	}
	return resources, nil
}

func virtualNetworks(ctx context.Context, a *azurerm, resourceType string, filters *filter.Filter) ([]provider.Resource, error) {
	virtualNetworks, err := a.azurer.ListVirtualNetworks(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list virtual networks from reader")
	}
	resources := make([]provider.Resource, 0, len(virtualNetworks))
	for _, virtualNetwork := range virtualNetworks {
		r := provider.NewResource(*virtualNetwork.ID, resourceType, a)
		// we set the name prior of reading it from the state
		// as it is required to able to List resources depending on this one
		if err := r.Data().Set("name", *virtualNetwork.Name); err != nil {
			return nil, errors.Wrapf(err, "unable to set name data on the provider.Resource for the virtual network '%s'", *virtualNetwork.Name)
		}
		resources = append(resources, r)
	}
	return resources, nil
}

func subnets(ctx context.Context, a *azurerm, resourceType string, filters *filter.Filter) ([]provider.Resource, error) {
	virtualNetworkNames, err := getVirtualNetworkNames(ctx, a, resourceType, filters)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list virtual networks from cache")
	}
	resources := make([]provider.Resource, 0)
	for _, virtualNetworkName := range virtualNetworkNames {
		subnets, err := a.azurer.ListSubnets(ctx, virtualNetworkName)
		if err != nil {
			return nil, errors.Wrap(err, "unable to list subnets from reader")
		}
		for _, subnet := range subnets {
			r := provider.NewResource(*subnet.ID, resourceType, a)
			resources = append(resources, r)
		}
	}
	return resources, nil
}

func networkInterfaces(ctx context.Context, a *azurerm, resourceType string, filters *filter.Filter) ([]provider.Resource, error) {
	networkInterfaces, err := a.azurer.ListInterfaces(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list network interfaces from reader")
	}
	resources := make([]provider.Resource, 0, len(networkInterfaces))
	for _, networkInterface := range networkInterfaces {
		r := provider.NewResource(*networkInterface.ID, resourceType, a)
		resources = append(resources, r)
	}
	return resources, nil
}

func networkSecurityGroups(ctx context.Context, a *azurerm, resourceType string, filters *filter.Filter) ([]provider.Resource, error) {
	securityGroups, err := a.azurer.ListSecurityGroups(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list network security groups from reader")
	}
	resources := make([]provider.Resource, 0, len(securityGroups))
	for _, securityGroup := range securityGroups {
		r := provider.NewResource(*securityGroup.ID, resourceType, a)
		resources = append(resources, r)
	}
	return resources, nil
}

func virtualMachineScaleSets(ctx context.Context, a *azurerm, resourceType string, filters *filter.Filter) ([]provider.Resource, error) {
	virtualMachineScaleSets, err := a.azurer.ListVirtualMachineScaleSets(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list virtual machines scale sets from reader")
	}
	resources := make([]provider.Resource, 0, len(virtualMachineScaleSets))
	for _, virtualMachineScaleSet := range virtualMachineScaleSets {
		r := provider.NewResource(*virtualMachineScaleSet.ID, resourceType, a)
		resources = append(resources, r)
	}
	return resources, nil
}

func virtualDesktopHostPools(ctx context.Context, a *azurerm, resourceType string, filters *filter.Filter) ([]provider.Resource, error) {
	pools, err := a.azurer.ListHostPools(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list host pools from reader")
	}
	resources := make([]provider.Resource, 0, len(pools))
	for _, hostPool := range pools {
		r := provider.NewResource(*hostPool.ID, resourceType, a)
		resources = append(resources, r)
	}
	return resources, nil
}

func virtualApplicationGroups(ctx context.Context, a *azurerm, resourceType string, filters *filter.Filter) ([]provider.Resource, error) {
	// the second argument; "filter" is set to "" because "Valid properties for filtering are applicationGroupType."
	// https://godoc.org/github.com/Azure/azure-sdk-for-go/services/preview/desktopvirtualization/mgmt/2019-12-10-preview/desktopvirtualization#ApplicationGroupsClient.ListByResourceGroup
	applicationGroups, err := a.azurer.ListApplicationGroups(ctx, "")
	if err != nil {
		return nil, errors.Wrap(err, "unable to list application groups from reader")
	}
	resources := make([]provider.Resource, 0, len(applicationGroups))
	for _, applicationGroup := range applicationGroups {
		r := provider.NewResource(*applicationGroup.ID, resourceType, a)
		resources = append(resources, r)
	}
	return resources, nil
}

func logicAppWorkflows(ctx context.Context, a *azurerm, resourceType string, filters *filter.Filter) ([]provider.Resource, error) {
	appWorkflows, err := a.azurer.ListWorkflows(ctx, nil, "")
	if err != nil {
		return nil, errors.Wrap(err, "unable to list logic app workflows from reader")
	}
	resources := make([]provider.Resource, 0, len(appWorkflows))
	for _, appWorkflow := range appWorkflows {
		r := provider.NewResource(*appWorkflow.ID, resourceType, a)
		// we set the name prior of reading it from the state
		// as it is required to able to List resources depending on this one
		if err := r.Data().Set("name", *appWorkflow.Name); err != nil {
			return nil, errors.Wrapf(err, "unable to set name data on the provider.Resource for the app workflow '%s'", *appWorkflow.Name)
		}
		resources = append(resources, r)
	}
	return resources, nil
}

func logicAppTriggerCustoms(ctx context.Context, a *azurerm, resourceType string, filters *filter.Filter) ([]provider.Resource, error) {
	appWorkflowNames, err := getWorkflowNames(ctx, a, resourceType, filters)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list logic app workflows from reader")
	}

	resources := make([]provider.Resource, 0)
	for _, appWorkflowName := range appWorkflowNames {
		triggers, err := a.azurer.ListWorkflowTriggers(ctx, appWorkflowName, nil, "")
		if err != nil {
			return nil, errors.Wrap(err, "unable to list logic app trigger HTTP requests from reader")
		}
		for _, trigger := range triggers {
			r := provider.NewResource(*trigger.ID, resourceType, a)
			resources = append(resources, r)
		}
	}
	return resources, nil
}

func logicAppActionCustoms(ctx context.Context, a *azurerm, resourceType string, filters *filter.Filter) ([]provider.Resource, error) {
	appWorkflowNames, err := getWorkflowNames(ctx, a, resourceType, filters)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list logic app workflows from reader")
	}

	resources := make([]provider.Resource, 0)
	for _, appWorkflowName := range appWorkflowNames {
		runs, err := a.azurer.ListWorkflowRuns(ctx, appWorkflowName, nil, "")
		if err != nil {
			return nil, errors.Wrap(err, "unable to list workflow runs from reader")
		}

		for _, run := range runs {
			actions, err := a.azurer.ListWorkflowRunActions(ctx, appWorkflowName, *run.Name, nil, "")
			if err != nil {
				return nil, errors.Wrap(err, "unable to list workflow run actions from reader")
			}
			for _, action := range actions {
				r := provider.NewResource(*action.ID, resourceType, a)
				resources = append(resources, r)
			}
		}
	}
	return resources, nil
}

func virtualMachineExtensions(ctx context.Context, a *azurerm, resourceType string, filters *filter.Filter) ([]provider.Resource, error) {
	virtualMachineNames, err := getVirtualMachineNames(ctx, a, resourceType, filters)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list virtual machines from reader")
	}

	resources := make([]provider.Resource, 0)
	for _, virtualMachineName := range virtualMachineNames {
		extensions, err := a.azurer.ListVirtualMachineExtensions(ctx, virtualMachineName, "")
		if err != nil {
			return nil, errors.Wrap(err, "unable to list virtual machine extensions from reader")
		}

		for _, extension := range extensions {
			r := provider.NewResource(*extension.ID, resourceType, a)
			resources = append(resources, r)
		}
	}
	return resources, nil
}

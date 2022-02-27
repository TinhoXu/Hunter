package core

import (
	"fmt"
	"github.com/Dreamacro/clash/adapter"
	"github.com/Dreamacro/clash/adapter/outbound"
	"github.com/Dreamacro/clash/adapter/outboundgroup"
	"github.com/Dreamacro/clash/adapter/provider"
	"github.com/Dreamacro/clash/common/structure"
	C "github.com/Dreamacro/clash/config"
	"github.com/Dreamacro/clash/constant"
	providerTypes "github.com/Dreamacro/clash/constant/provider"
	"github.com/Dreamacro/clash/log"
	"os"
)

// Parse config
func Parse(path string) (proxies map[string]constant.Proxy, providersMap map[string]providerTypes.ProxyProvider, err error) {
	buf, err := readConfig(path)
	if err != nil {
		return nil, nil, err
	}

	rawCfg, err := C.UnmarshalRawConfig(buf)
	if err != nil {
		return nil, nil, err
	}

	return parseProxies(rawCfg)
}

func readConfig(path string) ([]byte, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("configuration file %s is empty", path)
	}

	return data, err
}

func parseProxies(cfg *C.RawConfig) (proxies map[string]constant.Proxy, providersMap map[string]providerTypes.ProxyProvider, err error) {
	proxies = make(map[string]constant.Proxy)
	providersMap = make(map[string]providerTypes.ProxyProvider)
	proxiesConfig := cfg.Proxy
	groupsConfig := cfg.ProxyGroup
	providersConfig := cfg.ProxyProvider

	proxies["DIRECT"] = adapter.NewProxy(outbound.NewDirect())
	proxies["REJECT"] = adapter.NewProxy(outbound.NewReject())

	var proxyList []string
	proxyList = append(proxyList, "DIRECT", "REJECT")

	// parse proxy
	for idx, mapping := range proxiesConfig {
		proxy, err := adapter.ParseProxy(mapping)
		if err != nil {
			return nil, nil, fmt.Errorf("proxy %d: %w", idx, err)
		}

		if _, exist := proxies[proxy.Name()]; exist {
			return nil, nil, fmt.Errorf("proxy %s is the duplicate name", proxy.Name())
		}
		proxies[proxy.Name()] = proxy
		proxyList = append(proxyList, proxy.Name())
	}

	// keep the original order of ProxyGroups in config file
	for idx, mapping := range groupsConfig {
		groupName, existName := mapping["name"].(string)
		if !existName {
			return nil, nil, fmt.Errorf("proxy group %d: missing name", idx)
		}
		proxyList = append(proxyList, groupName)
	}

	// check if any loop exists and sort the ProxyGroups
	if err := proxyGroupsDagSort(groupsConfig); err != nil {
		return nil, nil, err
	}

	// parse and initial providers
	for name, mapping := range providersConfig {
		if name == provider.ReservedName {
			return nil, nil, fmt.Errorf("can not defined a provider called `%s`", provider.ReservedName)
		}

		pd, err := provider.ParseProxyProvider(name, mapping)
		if err != nil {
			return nil, nil, fmt.Errorf("parse proxy provider %s error: %w", name, err)
		}

		providersMap[name] = pd
	}

	for _, item := range providersMap {
		log.Infoln("Start initial provider %s", item.Name())
		if err := item.Initial(); err != nil {
			return nil, nil, fmt.Errorf("initial proxy provider %s error: %w", item.Name(), err)
		}
	}

	// parse proxy group
	for idx, mapping := range groupsConfig {
		group, err := outboundgroup.ParseProxyGroup(mapping, proxies, providersMap)
		if err != nil {
			return nil, nil, fmt.Errorf("proxy group[%d]: %w", idx, err)
		}

		groupName := group.Name()
		if _, exist := proxies[groupName]; exist {
			return nil, nil, fmt.Errorf("proxy group %s: the duplicate name", groupName)
		}

		proxies[groupName] = adapter.NewProxy(group)
	}

	// initial compatible provider
	for _, pd := range providersMap {
		if pd.VehicleType() != providerTypes.Compatible {
			continue
		}

		log.Debugln("Start initial compatible provider %s", pd.Name())
		if err := pd.Initial(); err != nil {
			return nil, nil, err
		}
	}

	var ps []constant.Proxy
	for _, v := range proxyList {
		ps = append(ps, proxies[v])
	}
	hc := provider.NewHealthCheck(ps, "", 0, true)
	pd, _ := provider.NewCompatibleProvider(provider.ReservedName, ps, hc)
	providersMap[provider.ReservedName] = pd

	global := outboundgroup.NewSelector(
		&outboundgroup.GroupCommonOption{
			Name: "GLOBAL",
		},
		[]providerTypes.ProxyProvider{pd},
	)
	proxies["GLOBAL"] = adapter.NewProxy(global)
	return proxies, providersMap, nil
}

// Check if ProxyGroups form DAG(Directed Acyclic Graph), and sort all ProxyGroups by dependency order.
// Meanwhile, record the original index in the config file.
// If loop is detected, return an error with location of loop.
func proxyGroupsDagSort(groupsConfig []map[string]interface{}) error {
	type graphNode struct {
		indegree int
		// topological order
		topo int
		// the original data in `groupsConfig`
		data map[string]interface{}
		// `outdegree` and `from` are used in loop locating
		outdegree int
		option    *outboundgroup.GroupCommonOption
		from      []string
	}

	decoder := structure.NewDecoder(structure.Option{TagName: "group", WeaklyTypedInput: true})
	graph := make(map[string]*graphNode)

	// Step 1.1 build dependency graph
	for _, mapping := range groupsConfig {
		option := &outboundgroup.GroupCommonOption{}
		if err := decoder.Decode(mapping, option); err != nil {
			return fmt.Errorf("ProxyGroup %s: %s", option.Name, err.Error())
		}

		groupName := option.Name
		if node, ok := graph[groupName]; ok {
			if node.data != nil {
				return fmt.Errorf("ProxyGroup %s: duplicate group name", groupName)
			}
			node.data = mapping
			node.option = option
		} else {
			graph[groupName] = &graphNode{0, -1, mapping, 0, option, nil}
		}

		for _, proxy := range option.Proxies {
			if node, ex := graph[proxy]; ex {
				node.indegree++
			} else {
				graph[proxy] = &graphNode{1, -1, nil, 0, nil, nil}
			}
		}
	}
	// Step 1.2 Topological Sort
	// topological index of **ProxyGroup**
	index := 0
	queue := make([]string, 0)
	for name, node := range graph {
		// in the beginning, put nodes that have `node.indegree == 0` into queue.
		if node.indegree == 0 {
			queue = append(queue, name)
		}
	}
	// every element in queue have indegree == 0
	for ; len(queue) > 0; queue = queue[1:] {
		name := queue[0]
		node := graph[name]
		if node.option != nil {
			index++
			groupsConfig[len(groupsConfig)-index] = node.data
			if len(node.option.Proxies) == 0 {
				delete(graph, name)
				continue
			}

			for _, proxy := range node.option.Proxies {
				child := graph[proxy]
				child.indegree--
				if child.indegree == 0 {
					queue = append(queue, proxy)
				}
			}
		}
		delete(graph, name)
	}

	// no loop is detected, return sorted ProxyGroup
	if len(graph) == 0 {
		return nil
	}

	// if loop is detected, locate the loop and throw an error
	// Step 2.1 rebuild the graph, fill `outdegree` and `from` filed
	for name, node := range graph {
		if node.option == nil {
			continue
		}

		if len(node.option.Proxies) == 0 {
			continue
		}

		for _, proxy := range node.option.Proxies {
			node.outdegree++
			child := graph[proxy]
			if child.from == nil {
				child.from = make([]string, 0, child.indegree)
			}
			child.from = append(child.from, name)
		}
	}
	// Step 2.2 remove nodes outside the loop. so that we have only the loops remain in `graph`
	queue = make([]string, 0)
	// initialize queue with node have outdegree == 0
	for name, node := range graph {
		if node.outdegree == 0 {
			queue = append(queue, name)
		}
	}
	// every element in queue have outdegree == 0
	for ; len(queue) > 0; queue = queue[1:] {
		name := queue[0]
		node := graph[name]
		for _, f := range node.from {
			graph[f].outdegree--
			if graph[f].outdegree == 0 {
				queue = append(queue, f)
			}
		}
		delete(graph, name)
	}
	// Step 2.3 report the elements in loop
	loopElements := make([]string, 0, len(graph))
	for name := range graph {
		loopElements = append(loopElements, name)
		delete(graph, name)
	}
	return fmt.Errorf("loop is detected in ProxyGroup, please check following ProxyGroups: %v", loopElements)
}

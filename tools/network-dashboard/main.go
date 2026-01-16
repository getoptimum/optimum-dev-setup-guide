package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type NodeHealth struct {
	Status     string `json:"status"`
	CPUUsed    string `json:"cpu_used"`
	MemoryUsed string `json:"memory_used"`
	DiskUsed   string `json:"disk_used"`
	Country    string `json:"country"`
}

type NodeState struct {
	PubKey    string   `json:"pub_key"`
	Peers     []string `json:"peers"`
	Addresses []string `json:"addresses"`
	Topics    []string `json:"topics"`
}

type ProxyHealth struct {
	Status     string `json:"status"`
	CPUUsed    string `json:"cpu_used"`
	MemoryUsed string `json:"memory_used"`
	DiskUsed   string `json:"disk_used"`
	Country    string `json:"country"`
}

type NodeCountries struct {
	Countries map[string]string `json:"countries"`
	Count     int               `json:"count"`
}

type NodeInfo struct {
	Name      string
	URL       string
	Health    *NodeHealth
	State     *NodeState
	Available bool
	Error     string
}

type ProxyInfo struct {
	Name      string
	URL       string
	Health    *ProxyHealth
	Available bool
	Error     string
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

func fetchJSON(url string, target interface{}) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, target)
}

func fetchNodeInfo(name, baseURL string) NodeInfo {
	info := NodeInfo{Name: name, URL: baseURL}

	health := &NodeHealth{}
	if err := fetchJSON(baseURL+"/api/v1/health", health); err != nil {
		info.Error = err.Error()
		return info
	}
	info.Health = health

	state := &NodeState{}
	if err := fetchJSON(baseURL+"/api/v1/node-state", state); err != nil {
		info.Error = err.Error()
		return info
	}
	info.State = state
	info.Available = true

	return info
}

func fetchProxyInfo(name, baseURL string) ProxyInfo {
	info := ProxyInfo{Name: name, URL: baseURL}

	health := &ProxyHealth{}
	if err := fetchJSON(baseURL+"/api/v1/health", health); err != nil {
		info.Error = err.Error()
		return info
	}
	info.Health = health
	info.Available = true

	return info
}

func printDashboard(nodes []NodeInfo, proxies []ProxyInfo, nodeCountries *NodeCountries) {
	fmt.Println(strings.Repeat("=", 100))
	fmt.Printf("%-50s %s\n", "mump2p NETWORK DASHBOARD", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println(strings.Repeat("=", 100))
	fmt.Println()

	if len(proxies) > 0 {
		fmt.Println("PROXIES")
		fmt.Println(strings.Repeat("-", 100))
		fmt.Printf("%-15s %-8s %-10s %-10s %-10s %-15s %-20s\n",
			"Name", "Status", "CPU %", "Memory %", "Disk %", "Country", "URL")
		fmt.Println(strings.Repeat("-", 100))
		for _, p := range proxies {
			status := "DOWN"
			cpu, mem, disk, country := "N/A", "N/A", "N/A", "N/A"
			if p.Available && p.Health != nil {
				status = p.Health.Status
				cpu = p.Health.CPUUsed
				mem = p.Health.MemoryUsed
				disk = p.Health.DiskUsed
				country = p.Health.Country
			}
			fmt.Printf("%-15s %-8s %-10s %-10s %-10s %-15s %-20s\n",
				p.Name, status, cpu, mem, disk, country, p.URL)
		}
		fmt.Println()
	}

	if len(nodes) > 0 {
		fmt.Println("P2P NODES")
		fmt.Println(strings.Repeat("-", 100))
		fmt.Printf("%-15s %-8s %-10s %-10s %-10s %-8s %-8s %-15s %-20s\n",
			"Name", "Status", "CPU %", "Memory %", "Disk %", "Peers", "Topics", "Country", "URL")
		fmt.Println(strings.Repeat("-", 100))
		for _, n := range nodes {
			status := "DOWN"
			cpu, mem, disk, country := "N/A", "N/A", "N/A", "N/A"
			peers, topics := "0", "0"
			if n.Available {
				if n.Health != nil {
					status = n.Health.Status
					cpu = n.Health.CPUUsed
					mem = n.Health.MemoryUsed
					disk = n.Health.DiskUsed
					country = n.Health.Country
				}
				if n.State != nil {
					peers = fmt.Sprintf("%d", len(n.State.Peers))
					topics = fmt.Sprintf("%d", len(n.State.Topics))
				}
			}
			fmt.Printf("%-15s %-8s %-10s %-10s %-10s %-8s %-8s %-15s %-20s\n",
				n.Name, status, cpu, mem, disk, peers, topics, country, n.URL)
		}
		fmt.Println()

		fmt.Println("NODE DETAILS")
		fmt.Println(strings.Repeat("-", 100))
		for _, n := range nodes {
			if !n.Available {
				fmt.Printf("%s: %s\n", n.Name, n.Error)
				continue
			}
			if n.State == nil {
				continue
			}
			fmt.Printf("%s (Peer ID: %s)\n", n.Name, n.State.PubKey)
			fmt.Printf("  Peers: %d\n", len(n.State.Peers))
			if len(n.State.Peers) > 0 {
				fmt.Printf("  Peer IDs: %s\n", strings.Join(n.State.Peers[:min(5, len(n.State.Peers))], ", "))
				if len(n.State.Peers) > 5 {
					fmt.Printf("  ... and %d more\n", len(n.State.Peers)-5)
				}
			}
			fmt.Printf("  Topics: %d", len(n.State.Topics))
			if len(n.State.Topics) > 0 {
				fmt.Printf(" [%s]\n", strings.Join(n.State.Topics, ", "))
			} else {
				fmt.Println()
			}
			fmt.Printf("  Addresses: %s\n", strings.Join(n.State.Addresses, ", "))
			fmt.Println()
		}
	}

	if nodeCountries != nil && nodeCountries.Count > 0 {
		fmt.Println("NODE COUNTRIES")
		fmt.Println(strings.Repeat("-", 100))
		countryCount := make(map[string]int)
		for _, country := range nodeCountries.Countries {
			countryCount[country]++
		}
		for country, count := range countryCount {
			fmt.Printf("%s: %d node(s)\n", country, count)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 100))
}

func main() {
	var (
		proxyURLsFlag = flag.String("proxies", "", "Comma-separated list of proxy URLs (e.g., http://localhost:8081,http://localhost:8082)")
		nodeURLsFlag  = flag.String("nodes", "", "Comma-separated list of node URLs (e.g., http://localhost:9091,http://localhost:9092)")
		proxyBase     = flag.String("proxy-base", "", "IP(s) or URL(s) for remote proxies - will prepend http:// and append :8080")
		nodeBase      = flag.String("node-base", "", "IP(s) or URL(s) for remote nodes (optional) - will prepend http:// and append :8081")
		local         = flag.Bool("local", false, "Use localhost defaults (proxies: 8081,8082; nodes: 9091-9094)")
	)
	flag.Parse()

	var proxies []ProxyInfo
	var nodes []NodeInfo

	if *local {
		proxyAddrs := []string{"http://localhost:8081", "http://localhost:8082"}
		for i, url := range proxyAddrs {
			proxies = append(proxies, fetchProxyInfo(fmt.Sprintf("proxy-%d", i+1), url))
		}
		nodeAddrs := []string{"http://localhost:9091", "http://localhost:9092", "http://localhost:9093", "http://localhost:9094"}
		for i, url := range nodeAddrs {
			nodes = append(nodes, fetchNodeInfo(fmt.Sprintf("p2pnode-%d", i+1), url))
		}
	} else if *proxyBase != "" {
		bases := strings.Split(*proxyBase, ",")
		for i, base := range bases {
			base = strings.TrimSpace(base)
			if base == "" {
				continue
			}
			if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
				base = "http://" + base
			}
			url := base + ":8080"
			proxies = append(proxies, fetchProxyInfo(fmt.Sprintf("proxy-%d", i+1), url))
		}
	} else if *proxyURLsFlag != "" {
		urls := strings.Split(*proxyURLsFlag, ",")
		for i, url := range urls {
			url = strings.TrimSpace(url)
			if url == "" {
				continue
			}
			proxies = append(proxies, fetchProxyInfo(fmt.Sprintf("proxy-%d", i+1), url))
		}
	}

	if *nodeBase != "" {
		bases := strings.Split(*nodeBase, ",")
		for i, base := range bases {
			base = strings.TrimSpace(base)
			if base == "" {
				continue
			}
			if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
				base = "http://" + base
			}
			url := base + ":8081"
			nodes = append(nodes, fetchNodeInfo(fmt.Sprintf("p2pnode-%d", i+1), url))
		}
	} else if *nodeURLsFlag != "" {
		urls := strings.Split(*nodeURLsFlag, ",")
		for i, url := range urls {
			url = strings.TrimSpace(url)
			if url == "" {
				continue
			}
			nodes = append(nodes, fetchNodeInfo(fmt.Sprintf("p2pnode-%d", i+1), url))
		}
	}

	if len(proxies) == 0 && len(nodes) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No proxies or nodes specified. Use -local, -proxy-base, or -proxies/-nodes flags.\n")
		flag.Usage()
		os.Exit(1)
	}

	var nodeCountries *NodeCountries
	if len(proxies) > 0 && proxies[0].Available {
		nc := &NodeCountries{}
		if err := fetchJSON(proxies[0].URL+"/api/v1/node-countries", nc); err == nil {
			nodeCountries = nc
		}
	}

	printDashboard(nodes, proxies, nodeCountries)
}

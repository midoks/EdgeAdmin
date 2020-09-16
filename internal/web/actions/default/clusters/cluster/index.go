package cluster

import (
	"encoding/json"
	"fmt"
	"github.com/TeaOSLab/EdgeAdmin/internal/configs/nodes"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeAdmin/internal/web/actions/actionutils"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"time"
)

type IndexAction struct {
	actionutils.ParentAction
}

func (this *IndexAction) Init() {
	this.Nav("", "node", "index")
	this.SecondMenu("nodes")
}

func (this *IndexAction) RunGet(params struct {
	ClusterId      int64
	InstalledState int
}) {
	this.Data["installState"] = params.InstalledState

	countResp, err := this.RPC().NodeRPC().CountAllEnabledNodesMatch(this.AdminContext(), &pb.CountAllEnabledNodesMatchRequest{
		ClusterId:    params.ClusterId,
		InstallState: types.Int32(params.InstalledState),
	})
	if err != nil {
		this.ErrorPage(err)
		return
	}

	page := this.NewPage(countResp.Count)
	this.Data["page"] = page.AsHTML()

	nodesResp, err := this.RPC().NodeRPC().ListEnabledNodesMatch(this.AdminContext(), &pb.ListEnabledNodesMatchRequest{
		Offset:       page.Offset,
		Size:         page.Size,
		ClusterId:    params.ClusterId,
		InstallState: types.Int32(params.InstalledState),
	})
	nodeMaps := []maps.Map{}
	for _, node := range nodesResp.Nodes {
		// 状态
		status := &nodes.NodeStatus{}
		if len(node.Status) > 0 && node.Status != "null" {
			err = json.Unmarshal([]byte(node.Status), &status)
			if err != nil {
				logs.Error(err)
				continue
			}
			status.IsActive = time.Now().Unix()-status.UpdatedAt < 120 // 2分钟之内认为活跃
		}

		// IP
		ipAddressesResp, err := this.RPC().NodeIPAddressRPC().FindAllEnabledIPAddressesWithNodeId(this.AdminContext(), &pb.FindAllEnabledIPAddressesWithNodeIdRequest{NodeId: node.Id})
		if err != nil {
			this.ErrorPage(err)
			return
		}
		ipAddresses := []maps.Map{}
		for _, addr := range ipAddressesResp.Addresses {
			ipAddresses = append(ipAddresses, maps.Map{
				"id":   addr.Id,
				"name": addr.Name,
				"ip":   addr.Ip,
			})
		}

		nodeMaps = append(nodeMaps, maps.Map{
			"id":          node.Id,
			"name":        node.Name,
			"isInstalled": node.IsInstalled,
			"installStatus": maps.Map{
				"isRunning":  node.InstallStatus.IsRunning,
				"isFinished": node.InstallStatus.IsFinished,
				"isOk":       node.InstallStatus.IsOk,
				"error":      node.InstallStatus.Error,
			},
			"status": maps.Map{
				"isActive":     status.IsActive,
				"updatedAt":    status.UpdatedAt,
				"hostname":     status.Hostname,
				"cpuUsage":     status.CPUUsage,
				"cpuUsageText": fmt.Sprintf("%.2f%%", status.CPUUsage*100),
				"memUsage":     status.MemoryUsage,
				"memUsageText": fmt.Sprintf("%.2f%%", status.MemoryUsage*100),
			},
			"cluster": maps.Map{
				"id":   node.Cluster.Id,
				"name": node.Cluster.Name,
			},
			"ipAddresses": ipAddresses,
		})
	}
	this.Data["nodes"] = nodeMaps

	this.Show()
}
package network

import (
	"docker/container"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
)

var (
	defaultNetworkPath = "/var/run/mydocker/network/network/"
	drivers            = map[string]NetworkDriver{}
	networks           = map[string]*Network{}
)

type Network struct {
	Name    string     // 网络名
	IpRange *net.IPNet // 地址段
	Driver  string     // 网络驱动名
}

type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	Network     *Network
	PortMapping []string
}

type NetworkDriver interface {
	// Name 驱动名
	Name() string
	// Create 创建网络
	Create(subnet string, name string) (*Network, error)
	Delete(network Network) error
	// Connect 连接容器网络端点到网络
	Connect(network *Network, endpoint *Endpoint) error
	// Disconnect 从网络上移除容器网络端点
	Disconnect(network Network, endpoint *Endpoint) error
}

func (nw *Network) dump(dumpPath string) error {
	if _, err := os.Stat(dumpPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(dumpPath, 0644)
		} else {
			return err
		}
	}

	// 保存的文件名是网络的名字
	nwPath := path.Join(dumpPath, nw.Name)
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("error：%v", err)
		return err
	}
	defer nwFile.Close()

	// 序列化网络对象到json的字符串
	nwJson, err := json.Marshal(nw)
	if err != nil {
		logrus.Errorf("error：%v", err)
		return err
	}

	_, err = nwFile.Write(nwJson)
	if err != nil {
		logrus.Errorf("error：%v", err)
		return err
	}
	return nil
}

func (nw *Network) remove(dumpPath string) error {
	if _, err := os.Stat(path.Join(dumpPath, nw.Name)); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	} else {
		return os.Remove(path.Join(dumpPath, nw.Name))
	}
}

func (nw *Network) load(dumpPath string) error {
	nwConfigFile, err := os.Open(dumpPath)
	defer nwConfigFile.Close()
	if err != nil {
		return err
	}
	nwJson := make([]byte, 2000)
	n, err := nwConfigFile.Read(nwJson)
	if err != nil {
		return err
	}

	err = json.Unmarshal(nwJson[:n], nw)
	if err != nil {
		logrus.Errorf("Error load nw info: %v", err)
		return err
	}
	return nil
}

func Init() error {
	// 加载网络驱动
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	// 判断网络的配置目录是否存在，不存在则创建
	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(defaultNetworkPath, 0644)
		} else {
			return err
		}
	}

	// 检查网络配置目录中的所有文件，遍历指定的path目录并执行第二个参数中的函数指针去处理目录下的每一个文件
	filepath.Walk(defaultNetworkPath, func(nwPath string, info os.FileInfo, err error) error {
		if strings.HasSuffix(nwPath, "/") {
			return nil
		}
		// 加载载文件名作为网络名
		_, nwName := path.Split(nwPath)
		nw := &Network{
			Name: nwName,
		}

		// 加载网络的配置信息
		if err := nw.load(nwPath); err != nil {
			logrus.Errorf("error load network: %v", err)
		}

		networks[nwName] = nw
		return nil
	})

	//logrus.Infof("networks: %v", networks)

	return nil
}

func CreateNetwork(driver, subnet, name string) error {
	// 将网段的字符串转换成net.IPNet的对象
	_, cidr, _ := net.ParseCIDR(subnet)
	// 通过IPAM分配网关IP，获取到网段中第一个IP作为网关的IP
	gatewayIp, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = gatewayIp

	// 调用指定的网络驱动创建网络
	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}

	// 保存网络信息，将网络的信息保存在文件系统中，以便查询和在网络上连接网络端点
	return nw.dump(defaultNetworkPath)
}

func ListNetwork() {
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIpRange\tDriver\n")
	for _, nw := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			nw.Name,
			nw.IpRange.String(),
			nw.Driver,
		)
	}
	if err := w.Flush(); err != nil {
		logrus.Errorf("Flush error %v", err)
		return
	}
}

func DeleteNetwork(networkName string) error {
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No Such Network: %s", networkName)
	}

	// 1. 释放IP
	if err := ipAllocator.Release(nw.IpRange, &nw.IpRange.IP); err != nil {
		return fmt.Errorf("Error Remove Network gateway ip: %v", err)
	}

	// 2. 删除网络设备
	if err := drivers[nw.Driver].Delete(*nw); err != nil {
		return fmt.Errorf("Error Remove Network DriverError: %v", err)
	}

	// 3. 删除网络对应的配置文件
	return nw.remove(defaultNetworkPath)
}

func enterContainerNetns(enLink *netlink.Link, cinfo *container.ContainerInfo) func() {
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.Pid), os.O_RDONLY, 0)
	if err != nil {
		logrus.Errorf("error get container net namespace, %v", err)
	}

	nsFD := f.Fd()
	// 锁定线程，保证一直在所需要的网络空间中
	runtime.LockOSThread()

	// 修改veth peer 另外一端移到容器的namespace中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		logrus.Errorf("error set link netns , %v", err)
	}

	// 获取当前的网络namespace
	origns, err := netns.Get()
	if err != nil {
		logrus.Errorf("error get current netns, %v", err)
	}

	// 设置当前进程到新的网络namespace，并在函数执行完成之后再恢复到之前的namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		logrus.Errorf("error set netns, %v", err)
	}
	return func() {
		netns.Set(origns)
		origns.Close()
		runtime.UnlockOSThread()
		f.Close()
	}
}

func configEndpointIpAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo) error {
	// 通过网络端点中“Veth”的另一端将容器的网络端点加入到容器的网络空间中
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}

	// 执行完函数后，恢复为默认的网络空间
	defer enterContainerNetns(&peerLink, cinfo)()

	interfaceIP := *ep.Network.IpRange
	interfaceIP.IP = ep.IPAddress

	// 设置容器内Veth端点的IP
	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%v,%s", ep.Network, err)
	}

	// 启动容器内的Veth端点
	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}

	// 启动本地回环地址
	if err = setInterfaceUP("lo"); err != nil {
		return err
	}

	// 设置容器内的外部请求都通过容器内的Veth端点访问
	_, cidr, err := net.ParseCIDR("0.0.0.0/0")
	if err != nil {
		return err
	}

	// 构建要添加的路由数据，包括网络设备、网关IP及目的网段
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IpRange.IP,
		Dst:       cidr,
	}

	// 调用netlink的RouteAdd，添加路由到容器的网络空间
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}

	return nil
}

func configPortMapping(ep *Endpoint, cinfo *container.ContainerInfo) error {
	for _, pm := range ep.PortMapping {
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			logrus.Errorf("port mapping format error, %v", pm)
			continue
		}
		// 在iptables的PREROUTING中添加DNAT规则，将宿主机的端口请求转发到容器的地址和端口上
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		output, err := cmd.Output()
		if err != nil {
			logrus.Errorf("iptables Output, %v", output)
			continue
		}
	}
	return nil
}

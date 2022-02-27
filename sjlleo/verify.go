package sjlleo

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/TinhoXu/Hunter/common"
	"github.com/TinhoXu/Hunter/tools"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func Verify(method string, custom string, address string) {
	var ipv4, ipv6 string
	var NextLineSignal = false

	flag.Parse()

	// 解析ip地址
	ns, err := net.LookupHost(common.NetflixDns)
	if err != nil {
		fmt.Printf("Err: %s", err.Error())
		return
	}

	switch {
	case len(ns) != 0:
		for _, n := range ns {

			if tools.ParseIP(n) == 4 {
				ipv4 = n
			}
			if tools.ParseIP(n) == 6 {
				ipv6 = "[" + n + "]"
			}

		}

	}

	shellPrinter(0)

	if method == "full" {
		fmt.Println("\033[0;35m模式：详细信息模式\033[0m")
	} else if custom == "" {
		fmt.Println("\033[0;35m模式：简洁信息模式\033[0m")
	} else {
		fmt.Println("\033[0;35m模式：自定义影片测试模式\033[0m")
	}

	// 拼接非自制剧的URL
	testURL := common.NetflixUrl + strconv.Itoa(common.NonSelfMadeAvailableID)
	ipv4CountryCode := requestIP(testURL, ipv4, address)
	ipv6CountryCode := requestIP(testURL, ipv6, address)

	/***
	 * 检查CountryCode返回值:
	 * Error 代表该网络访问失败
	 * Ban 代表无法解锁这个ID种类的影片
	 * 此处如果显示值不为Error则都应该继续检测
	***/
	if !strings.Contains(ipv4CountryCode, "Error") {
		// 开启换行信号,在IPv4检测完毕后换行
		NextLineSignal = true
		shellPrinter(3)
		// 检测是否为自定义测试模式
		if custom != "" {
			if tools.IsNumeric(custom) == false {
				fmt.Println("\033[0;34m您输入的不是数字！\033[0m")
				return
			} else {
				MovieID, _ := strconv.Atoi(custom)
				if unblockTest(MovieID, ipv4, address) {
					fmt.Println("\033[0;32m可以解锁此影片\033[0m")
				} else {
					fmt.Println("\033[0;31m不能解锁此影片\033[0m")
				}
			}

		} else {
			// 如果反馈为Ban，那么进一步检测是否支持Netflix地区解锁
			if strings.Contains(ipv4CountryCode, "Ban") {
				// 检测该IP所在的地区是否支持NF
				if unblockTest(common.AreaAvailableID, ipv4, address) {
					// 所在地区支持NF
					if method == "full" {
						shellPrinter(2)
						shellPrinter(7)
					}
					// 检测是否支持自制
					if unblockTest(common.SelfMadeAvailableID, ipv4, address) {
						// 支持自制剧
						if method == "full" {
							shellPrinter(5)
							shellPrinter(8)
							shellPrinter(10)
							fmt.Println("\n\033[1;34m判断结果：不支持Netflix解锁")
							testURL2 := common.NetflixUrl + strconv.Itoa(common.SelfMadeAvailableID)
							ipv4CountryCode2 := requestIP(testURL2, ipv4, address)
							fmt.Println("\033[0;36mNF库识别的IP地域信息：\033[1;36m" + tools.FindCountry(ipv4CountryCode2) + "区(" + strings.ToUpper(strings.Split(ipv4CountryCode2, "-")[0]) + ") Netflix 非原生IP\033[0m")
						} else {
							fmt.Println("\033[0;33m您的出口IP不能解锁Netflix，仅支持自制剧的观看\033[0m")
						}
					} else {
						// 不支持自制剧
						shellPrinter(6)
					}
				} else {
					// 所在地区不支持NF
					shellPrinter(1)
				}

			} else {
				// 如果支持非自制剧的解锁，则直接跳过自制剧的解锁
				if method == "full" {
					shellPrinter(2)
					shellPrinter(7)
					shellPrinter(5)
					shellPrinter(8)
					shellPrinter(9)
					fmt.Println("\n\033[1;34m判断结果：完整支持Netflix解锁")
				} else {
					fmt.Println("\033[0;32m您的出口IP完整解锁Netflix，支持非自制剧的观看\033[0m")
				}
				fmt.Println("\033[0;36m原生IP地域解锁信息：\033[1;36m" + tools.FindCountry(ipv4CountryCode) + "区(" + strings.ToUpper(strings.Split(ipv4CountryCode, "-")[0]) + ") Netflix 原生IP\033[0m")
			}
		}
	}

	if !strings.Contains(ipv6CountryCode, "Error") {
		// 如果存在在IPv4检测，那在其完毕后换行
		if NextLineSignal {
			fmt.Print("\n")
		}
		shellPrinter(4)
		// 判断是否为自定义检测状态
		if custom != "" {
			if tools.IsNumeric(custom) == false {
				fmt.Println("\033[0;34m您输入的不是数字！\033[0m")
				return
			} else {
				MovieID, _ := strconv.Atoi(custom)
				if unblockTest(MovieID, ipv6, address) {
					fmt.Println("\033[0;32m可以解锁此影片\033[0m")
				} else {
					fmt.Println("\033[0;31m不能解锁此影片\033[0m")
				}
			}
			return
		}
		// 如果反馈为Ban，那么进一步检测是否支持Netflix地区解锁
		if strings.Contains(ipv6CountryCode, "Ban") {
			// 检测该IP所在的地区是否支持NF
			if unblockTest(common.AreaAvailableID, ipv6, address) {
				// 所在地区支持NF
				if method == "full" {
					shellPrinter(2)
					shellPrinter(7)
				}
				// 检测是否支持自制
				if unblockTest(common.SelfMadeAvailableID, ipv6, address) {
					// 支持自制剧
					if method == "full" {
						shellPrinter(5)
						shellPrinter(8)
						shellPrinter(10)
						fmt.Println("\n\033[1;34m判断结果：不支持Netflix解锁")
						testURL62 := common.NetflixUrl + strconv.Itoa(common.SelfMadeAvailableID)
						ipv6CountryCode2 := requestIP(testURL62, ipv6, address)
						fmt.Println("\033[0;36mNF库识别的IP地域信息：\033[1;36m" + tools.FindCountry(ipv6CountryCode2) + "区(" + strings.ToUpper(strings.Split(ipv6CountryCode2, "-")[0]) + ") Netflix 非原生IP\033[0m")
					} else {
						fmt.Println("\033[0;33m您的出口IP不能解锁Netflix，仅支持自制剧的观看\033[0m")
					}
				} else {
					// 不支持自制剧
					shellPrinter(6)
				}
			} else {
				// 所在地区不支持NF
				shellPrinter(1)
			}

		} else {
			// 如果支持非自制剧的解锁，则直接跳过自制剧的解锁
			if method == "full" {
				shellPrinter(2)
				shellPrinter(7)
				shellPrinter(5)
				shellPrinter(8)
				shellPrinter(9)
				fmt.Println("\n\033[0;34m判断结果：完整支持Netflix解锁")
			} else {
				fmt.Println("\033[0;32m您的出口IP完整解锁Netflix，支持非自制剧的观看\033[0m")
			}
			fmt.Println("\033[0;36m原生IP地域解锁信息：\033[1;36m" + tools.FindCountry(ipv6CountryCode) + "区(" + strings.ToUpper(strings.Split(ipv6CountryCode, "-")[0]) + ") Netflix 原生IP\033[0m")
		}
	} else {
		if method == "full" {
			if NextLineSignal {
				fmt.Print("\n")
			}
			shellPrinter(4)
			fmt.Println("\033[0;31m本机不支持IPv6的访问\033[0m")
		}
	}
	fmt.Println("\n\033[0;36m感谢每一个正在使用本脚本的你，祝解锁Netflix！\033[0m")
}

func requestIP(reqUrl string, ip string, address string) string {
	if ip == "" {
		return "Error"
	}
	urlValue, err := url.Parse(reqUrl)
	if err != nil {
		return "Error"
	}
	host := urlValue.Host
	if ip == "" {
		ip = host
	}
	newReqUrl := strings.Replace(reqUrl, host, ip, 1)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{ServerName: host},
			// goodryb pull
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				LocalAddr: &net.TCPAddr{
					IP: net.ParseIP(address),
				},
			}).DialContext,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
		Timeout:       5 * time.Second,
	}
	req, err := http.NewRequest(http.MethodGet, newReqUrl, nil)
	if err != nil {
		// return errors.New(strings.ReplaceAll(err.Error(), newReqUrl, reqUrl))
		return "Error"
	}
	req.Host = host
	req.Header.Set("USER-AGENT", common.UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		// return errors.New(strings.ReplaceAll(err.Error(), newReqUrl, reqUrl))
		return "Error"
	}
	defer resp.Body.Close()

	Header := resp.Header

	if Header["X-Robots-Tag"] != nil {
		if Header["X-Robots-Tag"][0] == "index" {
			return "us"
		}
	}

	if Header["Location"] == nil {
		return "Ban"
	} else {
		return strings.Split(Header["Location"][0], "/")[3]
	}
}

func unblockTest(MoiveID int, ip string, address string) bool {
	testURL := common.NetflixUrl + strconv.Itoa(MoiveID)
	reCode := requestIP(testURL, ip, address)
	return !strings.Contains(reCode, "Ban")
}

func shellPrinter(Num int) {
	switch Num {
	case 0:
		fmt.Println("** Netflix 解锁检测小工具 v2.61 By \033[1;36m@sjlleo\033[0m **")
	case 1:
		fmt.Println("\033[0;33mNetflix不为您测试的出口IP提供服务\033[0m")
	case 2:
		fmt.Println("\033[0;32mNetflix在您测试的出口IP所在的地区提供服务，宽松版权的自制剧可以解锁\033[0m")
	case 3:
		fmt.Println("\033[0;36m[IPv4测试]\033[0m")
	case 4:
		fmt.Println("\033[0;36m[IPv6测试]\033[0m")
	case 5:
		fmt.Println("\033[0;32m支持解锁全部的自制剧\033[0m")
	case 6:
		fmt.Println("\033[0;31m不支持解锁带有强版权的自制剧\033[0m")
	case 7:
		fmt.Println("->> 正在检查是否完整支持自制剧 <<-")
	case 8:
		fmt.Println("->> 正在检查支持的Netflix地区 <<-")
	case 9:
		fmt.Println("\033[0;32m支持解锁非自制剧\033[0m")
	case 10:
		fmt.Println("\033[0;31m不支持解锁非自制剧\033[0m")
	}
}

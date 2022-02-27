package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Dreamacro/clash/common/batch"
	"github.com/TinhoXu/Hunter/core"
	"github.com/TinhoXu/Hunter/sjlleo"
	"github.com/TinhoXu/Hunter/tools"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"time"
)

var clashCfgPath = flag.String("path", "", "clash 配置文件路径，默认为当前目录下的 clash.yaml")
var method = flag.String("method", "", "模式选择(full/lite)")
var custom = flag.String("custom", "", "自定义测试NF影片ID\n绝命毒师的ID是 70143836")
var address = flag.String("address", "", "本机公网IP")

const ConfigFilename = "config.yaml"
const ClashFilename = "clash.yaml"

type Config struct {
	Path string `yaml:"path"`
}

func main() {
	flag.Parse()

	homeDir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		sjlleoVerify()
		return
	}

	configPath := path.Join(homeDir, ConfigFilename)
	defaultClashPath := path.Join(homeDir, ClashFilename)
	propPath := *clashCfgPath

	var config Config
	if propPath != "" {
		// 保存 propPath 到 config.yaml 文件
		if propPath == "." {
			config = Config{Path: ""}
		} else {
			config = Config{Path: propPath}
		}

		err = writeConfig(configPath, &config)
		if err != nil {
			fmt.Println(err)
			sjlleoVerify()
			return
		}
	}

	if exist, _ := tools.PathExists(configPath); exist && propPath == "" {
		// 如果 propPath 为空，则尝试读取配置
		config, err = readConfig(configPath)
		if err != nil {
			fmt.Println(err)
			sjlleoVerify()
			return
		}
		propPath = config.Path
	}

	verify(propPath, defaultClashPath)
}

func verify(propPath, defaultClashPath string) {
	if propPath == "" || propPath == "." {
		// 如果 propPath 为 "."，则用默认值
		propPath = defaultClashPath
	}

	fmt.Printf("clash.yaml pwd = %s\n", propPath)
	proxies, _, err := core.Parse(propPath)
	if err != nil {
		fmt.Println(err)
		sjlleoVerify()
		return
	}
	// fmt.Println(proxies)
	// fmt.Println(providers)

	b, _ := batch.New(context.Background(), batch.WithConcurrencyNum(10))
	defaultURLTestTimeout := time.Second * 5

	var resultMap = make(map[string][]string)
	for _, proxy := range proxies {
		p := proxy
		if p.Addr() == "" {
			continue
		}

		b.Go(p.Name(), func() (interface{}, error) {
			ctx, cancel := context.WithTimeout(context.Background(), defaultURLTestTimeout)
			defer cancel()

			var result string
			if *custom == "" {
				result = core.UnlockTest(ctx, p)
			} else {
				MovieID, _ := strconv.Atoi(*custom)
				if core.UnlockTestWithMovieID(ctx, p, MovieID) {
					result = core.Available
				} else {
					result = core.Unavailable
				}
			}

			resultMap[result] = append(resultMap[result], p.Name())
			fmt.Printf("name: %-30s, type: %-20s, result: %s \n", p.Name(), p.Type(), result)
			return nil, nil
		})
	}
	b.Wait()

	if len(resultMap) == 0 {
		fmt.Println("解析失败，请确认：\n    1.配置文件是否存在\n    2.配置文件是否配置正确")
		sjlleoVerify()
		return
	}

	fmt.Println("\033[0;36m------------------------------ 结果汇总 ------------------------------\033[0;36m")
	for key, value := range resultMap {
		sort.Strings(value)
		fmt.Println(key)
		for _, item := range value {
			fmt.Printf("\t%s\n", item)
		}
		fmt.Println()
	}
}

func sjlleoVerify() {
	fmt.Println("将完全使用 sjlleo/netflix-verify 校验当前网络的解锁情况")
	sjlleo.Verify(*method, *custom, *address)
}

func writeConfig(filename string, config *Config) (err error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return
	}

	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return
	}
	return
}

func readConfig(filename string) (config Config, err error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	// fmt.Println(string(content))
	err = yaml.Unmarshal(content, &config)
	return
}

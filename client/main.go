package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type AutoUpdate struct {
	LastVersion string
	Version     []struct {
		Number string
		Notes  string
		Files  []struct {
			Name string
			URL  string
		}
	}
}

const URL = "http://127.0.0.1:9527/version"

var autoupdate AutoUpdate

func main() {
	fmt.Println("开始检查更新...")
	err := checkUpdate()
	if err != nil {
		fmt.Println("检查更新出错")
		return
	}
	data, err := os.ReadFile("version.txt")
	if err != nil {
		fmt.Println("读取版本文件出错,请在程序目录运行")
		return
	}
	curVersion := string(data)

	if curVersion == autoupdate.LastVersion {
		fmt.Println("已经是最新版本")
		return
	}

	flag := false
	for _, v := range autoupdate.Version {
		if v.Number == curVersion {
			flag = true
			break
		}
	}

	if !flag {
		fmt.Println("当前版本太旧，请下载最新版本!!!")
		return
	}

	sort.Slice(autoupdate.Version, func(i int, j int) bool {
		return compareVersions(autoupdate.Version[i].Number, autoupdate.Version[j].Number) < 0
	})

	os.MkdirAll("updates", 0755)

	startUpdate := false
	for _, v := range autoupdate.Version {

		if v.Number == curVersion {
			startUpdate = true
			continue
		}
		if !startUpdate {
			continue
		}

		fmt.Println("发现需要更新的版本:", v.Number)
		fmt.Println("版本描述:", v.Notes)
		fmt.Printf("共需要更新 %d 个文件\n", len(v.Files))
		var w sync.WaitGroup
		for i, f := range v.Files {
			w.Add(1)
			go func(i int, name, url string) {
				defer w.Done()
				err := downloadAndUnzip(i, name, url)
				if err != nil {
					fmt.Printf("更新出错. 错误:%s 版本:%s,文件:%s\n", err.Error(), v.Number, url)
					fmt.Println("请尝试重试或联系管理员!!!")
					os.Exit(-1)
				}
			}(i+1, f.Name, f.URL)
		}
		w.Wait()
		os.WriteFile("version.txt", []byte(v.Number), 0766)
	}
	fmt.Println("更新完成!!!")
}

func downloadAndUnzip(i int, name, url string) error {
	fmt.Printf("正在更新第 %d 个文件...\n", i)

	rsp, err := http.Get(url)
	if err != nil {
		return err
	}
	if rsp.StatusCode != 200 {
		return errors.New("下载出错")
	}
	defer rsp.Body.Close()

	// 创建本地文件
	out, err := os.Create("updates/" + name)
	if err != nil {
		return err
	}
	//defer out.Close()

	fmt.Printf("开始下载第 %d 个文件\n", i)
	// 将 HTTP 响应体复制到本地文件
	_, err = io.Copy(out, rsp.Body)
	if err != nil {
		return err
	}
	out.Close()
	fmt.Printf("下载第 %d 个文件完成，开始解压\n", i)

	// 指定解压目录
	extractDir := "./"

	err = unzip("updates/"+name, extractDir)
	if err != nil {
		fmt.Printf("解压文件失败,错误: %s\n", err.Error())
		return err
	}

	fmt.Printf("第 %d 个文件更新完成\n", i)
	return nil
}

func checkUpdate() error {
	// 获取最新版本号
	rsp, err := http.Get(URL)
	if err != nil {
		return err
	}
	if rsp.StatusCode != 200 {
		return errors.New("请求出错")
	}

	defer rsp.Body.Close()
	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &autoupdate)

	if err != nil {
		return err
	}
	return nil
}

// 比较版本号的函数
func compareVersions(v1, v2 string) int {
	// 分割版本号字符串为切片
	ver1 := strings.Split(v1, ".")
	ver2 := strings.Split(v2, ".")

	// 比较主版本号
	if major1, major2 := compareSegment(ver1[0], ver2[0]); major1 != major2 {
		return major1 - major2
	}

	// 比较次版本号
	if minor1, minor2 := compareSegment(ver1[1], ver2[1]); minor1 != minor2 {
		return minor1 - minor2
	}

	// 比较修订版本号
	if patch1, patch2 := compareSegment(ver1[2], ver2[2]); patch1 != patch2 {
		return patch1 - patch2
	}

	return 0
}

// 比较版本号的单个段
func compareSegment(seg1, seg2 string) (int, int) {
	v1 := atoi(seg1)
	v2 := atoi(seg2)

	if v1 < v2 {
		return -1, 1
	} else if v1 > v2 {
		return 1, -1
	}

	return 0, 0
}

// 将字符串转换为整数
func atoi(s string) int {
	result := 0
	for _, c := range s {
		result = result*10 + int(c-'0')
	}
	return result
}

// 解压 ZIP 文件
func unzip(src, dest string) error {
	// 打开 ZIP 文件
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// 创建解压目录
	os.MkdirAll(dest, os.ModePerm)

	// 遍历 ZIP 文件中的文件
	for _, file := range r.File {
		// 打开 ZIP 文件中的每个文件
		rc, err := file.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		// 拼接解压路径
		filePath := filepath.Join(dest, file.Name)

		// 如果是目录，创建目录
		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		// 创建文件
		fileDir := filepath.Dir(filePath)
		os.MkdirAll(fileDir, os.ModePerm)
		f, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer f.Close()

		// 将 ZIP 文件中的内容复制到解压文件中
		_, err = io.Copy(f, rc)
		if err != nil {
			return err
		}
	}

	return nil
}

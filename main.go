package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/xuri/excelize/v2"

	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

const s3Endpoint = "https://s3.bitiful.net"

// Config 用于映射 config.yml 文件的内容
type Config struct {
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket_name"`
}

// Init 从配置文件读取信息并赋值给传入的变量
func Init(access_key *string, secret_key *string, bucket *string) {
	data, err := os.ReadFile("config.yml")
	if err != nil {
		log.Fatalf("无法读取配置文件: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}

	*access_key = cfg.AccessKey
	*secret_key = cfg.SecretKey
	*bucket = cfg.Bucket
}

func main() {

	var (
		s3AccessKey string
		s3SecretKey string
		bucket      string
	)

	// 读取配置文件
	Init(&s3AccessKey, &s3SecretKey, &bucket)

	fmt.Println("变量初始化完成——正在加载内容")

	folderName := "/" // 开始的文件夹名称 默认从根目录
	s3Client, err := getS3Client(s3AccessKey, s3SecretKey)

	if err != nil {
		log.Printf("err")
	}

	for true {
		// 获取桶内文件列表
		listObjsResponse, err := s3Client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
			Bucket:    aws.String(bucket),
			Prefix:    aws.String(folderName), // 设置 Prefix 为文件夹路径
			Delimiter: aws.String("/"),
			MaxKeys:   aws.Int32(50),
		})

		folderNames := make([]string, 0)

		if err != nil {
			log.Println("列出文件失败 检查配置, err=", err)
			return
		}
		fmt.Println("\n桶中的文件夹：")

		fmt.Println("---------------------------")
		for _, prefix := range listObjsResponse.CommonPrefixes {
			// fmt.Println(*prefix.Prefix)
			folderNames = append(folderNames, *prefix.Prefix)
		}

		fmt.Printf("请输入编号进入相应的文件夹：\n")
		for i, sr := range folderNames {
			fmt.Printf("[%d] : %s\n", i, sr)
		}
		filesLink := make([]string, 0)
		filesName := make([]string, 0)
		fmt.Println("\n下面是桶中的文件:", folderName)
		for _, object := range listObjsResponse.Contents {
			if *object.Key == folderName {
				continue
			}
			// fmt.Printf("https://testdata.s3.bitiful.net/\n", SetUrl(*object.Key))
			fmt.Printf("%s\n", *object.Key)
			filesLink = append(filesLink, "https://testdata.s3.bitiful.net/"+SetUrl(*object.Key))
			filesName = append(filesName, *object.Key)
		}
		fmt.Println("\n 输入 -1  导出当文件夹下文件的 Excel 表格:")

		var id int
		fmt.Scan(&id)
		if id < 0 {
			GetExcel(filesName, filesLink)
			break
		} else {
			//定位文件夹
			folderName = folderNames[id]
			fmt.Printf("%s", folderName)
		}
	}

	log.Println("list objects success")
}

// 获取S3客户端
func getS3Client(key, secret string) (*s3.Client, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == "S3" {
			return aws.Endpoint{
				URL: s3Endpoint,
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown service requested")
	})

	customProvider := credentials.NewStaticCredentialsProvider(key, secret, "")
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(customProvider), config.WithEndpointResolverWithOptions(customResolver))
	if err != nil {
		return nil, err
	}
	cfg.Region = "cn-east-1"
	s3client := s3.NewFromConfig(cfg)
	return s3client, nil
}
func SetUrl(key string) string {
	key = url.QueryEscape(key)
	// 将 + 替换为空格
	key = strings.ReplaceAll(key, "+", "%20")
	return key
}
func GetExcel(filesName []string, filesLink []string) {
	fmt.Printf("生成excel 表格文件\n\n")
	fmt.Println("---------------------------")

	// 检查是否有文件
	if len(filesName) == 0 {
		fmt.Println("当前目录下没有文件，无法生成Excel。")
		return
	}

	// 创建一个新的Excel文件
	f := excelize.NewFile()

	// 获取默认的第一个工作表
	sheetName := f.GetSheetName(0) // 使用索引0获取第一个工作表

	// 写入标题行
	f.SetCellValue(sheetName, "A1", "Name")
	f.SetCellValue(sheetName, "B1", "Link")

	// 写入数据行，从第二行开始
	for i := range filesName {
		row := i + 2 // 数据从第二行开始
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), GetFileName(filesName[i]))
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), filesLink[i])
		fmt.Printf("%s %s\n", filesName[i], filesLink[i])
	}

	// 保存文件
	filePath := "./SaveOK.xlsx"
	if err := f.SaveAs(filePath); err != nil {
		fmt.Println("保存文件时出错:", err)
		return
	}

	fmt.Println("Excel文件已生成:", filePath)
}

func GetFileName(fileName string) string {
	index := strings.LastIndex(fileName, "/")

	if index == -1 {
		return fileName
	}

	return fileName[index+1:]
}

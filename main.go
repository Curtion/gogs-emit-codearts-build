package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/region"
	art "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/codeartsbuild/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/codeartsbuild/v3/model"
)

var (
	config  Config
	client  *art.CodeArtsBuildClient
	jobsing = []Jobsing{}
	mtux    = &sync.Mutex{}
)

func init() {
	if _, err := os.Stat("config.toml"); err != nil {
		log.Fatal("config.toml文件不存在")
	}
	_, err := toml.DecodeFile("config.toml", &config)
	if err != nil {
		log.Fatal(err)
	}

	auth, err := basic.NewCredentialsBuilder().
		WithAk(config.Huawei.AK).
		WithSk(config.Huawei.SK).
		SafeBuild()
	if err != nil {
		log.Fatal(err)
	}

	artRegion := region.NewRegion(config.Huawei.Region, config.Huawei.EndPoint)

	artClient, err := art.CodeArtsBuildClientBuilder().
		WithRegion(artRegion).
		WithCredential(auth).
		SafeBuild()
	if err != nil {
		log.Fatal(err)
	}
	client = art.NewCodeArtsBuildClient(artClient)
}

func main() {
	http.HandleFunc("/hook", helloHandler)

	fmt.Println("服务启动成功: 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "读取Body失败", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	var data GogsWebhook
	if err := json.Unmarshal(body, &data); err != nil {
		http.Error(w, "请求体JSON解析失败, 中止", http.StatusInternalServerError)
		return
	}
	log.Printf("收到请求Repo:%s, Branch:%s\n", data.Repository.FullName, data.Ref)
	for _, job := range config.Jobs {
		if job.Repo == data.Repository.FullName && "refs/heads/"+job.Branch == data.Ref {
			stopOtherJob()
			log.Printf("执行任务:%s\n", job.Name)
			go run(job.Name, job.Branch)
		}
	}
	fmt.Fprint(w, "OK!")
}

func stopOtherJob() {
	mtux.Lock()
	defer mtux.Unlock()
	for index, job := range jobsing {
		status, err := getJobStatus(job.JobId)
		if err != nil {
			log.Println(err)
			continue
		}
		if status {
			if err := stopJob(job.JobId, job.Number); err != nil {
				log.Println(err)
			} else {
				jobsing = append(jobsing[:index], jobsing[index+1:]...)
				log.Printf("停止任务:%s, 编号:%d\n", job.JobId, job.Number)
			}
		} else {
			jobsing = append(jobsing[:index], jobsing[index+1:]...)
		}

	}
}

func run(name string, branch string) {
	id, err := getJobIdByName(name)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("任务ID:%s\n", id)
	numberStr, err := runJob(id, branch)
	if err != nil {
		log.Println(err)
		return
	}
	number, err := strconv.ParseInt(numberStr, 10, 32)
	if err != nil {
		log.Println(err)
		return
	}
	time.Sleep(5 * time.Second)

	status, err := getJobStatus(id)
	if err != nil {
		log.Println(err)
		return
	}
	if status {
		log.Println("任务执行成功")
		mtux.Lock()
		jobsing = append(jobsing, Jobsing{
			Number: int32(number),
			JobId:  id,
		})
		mtux.Unlock()
	}
}

func getJobIdByName(name string) (string, error) {
	request := &model.ShowJobListByProjectIdRequest{}
	request.PageIndex = int32(0)
	request.PageSize = int32(100)
	request.ProjectId = "ec92bf3022ec42b3b04c30c73d81f23a"
	response, err := client.ShowJobListByProjectId(request)
	if err == nil {
		for _, job := range *response.Jobs {
			if *job.JobName == name {
				return *job.Id, nil
			}
		}
		return "", errors.New("job not found")
	} else {
		return "", err
	}
}

func runJob(jobId string, branch string) (string, error) {
	request := &model.RunJobRequest{}
	var listParameterbody = []model.ParameterItem{
		{
			Name:  "codeBranch",
			Value: branch,
		},
	}
	request.Body = &model.RunJobRequestBody{
		Parameter: &listParameterbody,
		JobId:     jobId,
	}
	response, err := client.RunJob(request)
	if err == nil {
		if response.HttpStatusCode != 200 {
			return "", fmt.Errorf("运行任务失败: %d", response.HttpStatusCode)
		}
		return *response.ActualBuildNumber, nil
	}
	return "", err
}

func stopJob(jobId string, number int32) error {
	request := &model.StopBuildJobRequest{}
	request.JobId = jobId
	request.BuildNo = number
	_, err := client.StopBuildJob(request)
	return err
}

func getJobStatus(id string) (bool, error) {
	request := &model.ShowJobStatusRequest{}
	request.JobId = id
	response, err := client.ShowJobStatus(request)
	if err == nil {
		if response.HttpStatusCode != 200 {
			return false, fmt.Errorf("查询任务状态失败: %d", response.HttpStatusCode)
		}
		if *response.Result {
			return true, nil
		} else {
			return false, nil
		}
	}
	return false, err
}

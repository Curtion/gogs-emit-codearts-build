package main

type Config struct {
	Huawei HuaweiConfig
	Jobs   []JobConfig
}

type HuaweiConfig struct {
	AK       string
	SK       string
	Region   string
	EndPoint string
}

type JobConfig struct {
	Repo   string
	Name   string
	Branch string
}

type GogsWebhook struct {
	Ref        string
	Repository struct {
		FullName string `json:"full_name"`
	}
}

type Jobsing struct {
	Number  int32
	JobName string
	JobId   string
}

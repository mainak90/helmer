package models

type Chart struct {
	ID     	int    `json:id`
	Name  	string `json:name`
	Version string `json:version`
	Path   	string `json:path`
}

type Deploy struct {
	ID 			int	   				`json:id`
	Name		string 				`json:name`
	Chart		string 				`json:chart`
	Version		string 				`json: version`
	Namespace 	string				`json:namespace`
	Vars 		[]string 			`json:vars`
	Time  		int64					`json:time`
	Status  	string				`json:status`
	//"vars": ["mysqlRootPassword=admin@123,persistence.enabled=false,imagePullPolicy=Always"]
}
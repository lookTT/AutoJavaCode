package Model

type AutoGenerateJavaCodeConfig struct {
	Workspace WorkspaceConfig
	DirStruct []DirStructConfig
	MySql     MySqlConfig

	//重新赋值的字段
	CurStructPath    string
	CurStructPackage string
	CurMapperPath    string
	CurMapperPackage string
	CurXmlPath       string
}

type WorkspaceConfig struct {
	DirName         string
	DBName          string
	TableName       []string
	ForceCover      bool
	Author          string
	ClassNamePrefix string
	ClassNameSuffix string
	TemplateStruct  string
	TemplateMapper  string
	TemplateMybatis string
}

type DirStructConfig struct {
	Name          []string
	DBName        string
	StructPath    string
	StructPackage string
	MapperPath    string
	MapperPackage string
	XmlPath       string
}

type MySqlConfig struct {
	HOST              string
	POST              string
	UserName          string
	PassWd            string
	DBName            string
	IsDBNameInMyBatis bool
	TypeTranslate     map[string]string
}

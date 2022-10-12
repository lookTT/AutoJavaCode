package Model

type SStructTemplate struct {
	PACKAGE   string
	AUTHOR    string
	DATE      string
	CLASSNAME string
	DATA      string
}

type SMapperTemplate struct {
	PackageMapper string
	PackageStruct string
	AUTHOR        string
	DATE          string
	InterfaceName string
	StructName    string
}

type SMybatisTemplate struct {
	DBName                string //库名
	TableName             string //表名
	MapperPath            string //mapper路径
	StructPath            string //结构体路径
	CustomResultMap       string //结构体字段信息
	TableColumns          string //MySQL内字段信息
	EntityProperties      string //结构体内字段信息
	BatchEntityProperties string //批量操作结构体内字段信息
	UpdateContent         string //更新信息
	LimitContent          string //限定条件
}

type SFieldInfo struct {
	FieldName      string
	FieldNameCamel string
	FieldType      string
	FieldComment   string
}

package main

import (
	"AutoGenerateJavaCode/Common"
	"AutoGenerateJavaCode/Model"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
)

var db *sql.DB
var config Model.AutoGenerateJavaCodeConfig //配置文件
var structTemplate Model.SStructTemplate
var mapperTemplate Model.SMapperTemplate
var mybatisTemplate Model.SMybatisTemplate
var ttTemplate *template.Template

func initTemplate() {
	strTemplate, err := ioutil.ReadFile(config.Workspace.TemplateStruct)
	Common.CheckErr(err)
	ttTemplate, err = template.New(config.Workspace.TemplateStruct).Parse(string(strTemplate))
	Common.CheckErr(err)

	strTemplate, err = ioutil.ReadFile(config.Workspace.TemplateMapper)
	Common.CheckErr(err)
	ttTemplate, err = ttTemplate.New(config.Workspace.TemplateMapper).Parse(string(strTemplate))
	Common.CheckErr(err)

	strTemplate, err = ioutil.ReadFile(config.Workspace.TemplateMybatis)
	Common.CheckErr(err)
	ttTemplate, err = ttTemplate.New(config.Workspace.TemplateMybatis).Parse(string(strTemplate))
	Common.CheckErr(err)

	now := time.Now()
	strDate := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	structTemplate.PACKAGE = config.CurStructPackage
	structTemplate.DATE = strDate
	structTemplate.AUTHOR = config.Workspace.Author

	mapperTemplate.PackageMapper = config.CurMapperPackage
	//mapperTemplate.StructPackage = config.StructPackage
	mapperTemplate.AUTHOR = config.Workspace.Author
	mapperTemplate.DATE = strDate
}

// 初始化数据库连接
func initDB(config *Model.AutoGenerateJavaCodeConfig) error {
	urlPath := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", config.MySql.UserName, config.MySql.PassWd, config.MySql.HOST, config.MySql.POST, config.MySql.DBName)
	var err error
	db, err = sql.Open("mysql", urlPath)

	Common.CheckErr(err)
	db.SetMaxOpenConns(6000)
	db.SetMaxIdleConns(500)

	return nil
}

// 小驼峰
func HandlingStringsLittle(in string) string {
	tmp := HandlingStringsBig(in)
	return fmt.Sprintf("%s%s", strings.ToLower(tmp[0:1]), tmp[1:])
}

// 大驼峰
func HandlingStringsBig(in string) string {
	if in == "" {
		return in
	}
	var out bytes.Buffer
	sss := strings.Split(in, "_")
	var tmp string
	for _, s := range sss {
		tmp = s[0:1]
		out.WriteString(strings.ToUpper(tmp))
		out.WriteString(s[1:])
	}
	return out.String()
}

// 处理结构体
func StructHandler(curTableName string, sFieldInfos []Model.SFieldInfo, size int) {
	var sFieldInfo Model.SFieldInfo
	var strLine, DATA string
	for i := 0; i < size; i++ {
		sFieldInfo = sFieldInfos[i]
		if _, ok := config.MySql.TypeTranslate[sFieldInfo.FieldType]; !ok {
			continue
		}
		strLine = config.MySql.TypeTranslate[sFieldInfo.FieldType]

		strLine = fmt.Sprintf(strLine, sFieldInfo.FieldNameCamel, sFieldInfo.FieldComment)
		DATA += strLine
	}
	structTemplate.CLASSNAME = HandlingStringsBig(curTableName)
	structTemplate.CLASSNAME = fmt.Sprintf("%s%s%s", config.Workspace.ClassNamePrefix, structTemplate.CLASSNAME, config.Workspace.ClassNameSuffix)
	if len(DATA) > 2 {
		DATA = DATA[:len(DATA)-2]
	}
	structTemplate.DATA = DATA

	//创建目录
	Common.CheckErr(Common.MkdirAll(config.CurStructPath))
	//写文件
	tempPath := fmt.Sprintf("./%s/%s.java", config.CurStructPath, HandlingStringsBig(structTemplate.CLASSNAME))
	if !config.Workspace.ForceCover && Common.FileExists(tempPath) {
		//不覆盖
		tempPath = fmt.Sprintf("./%s/%s-copy.java", config.CurStructPath, HandlingStringsBig(structTemplate.CLASSNAME))
	}
	f, err := os.Create(tempPath)
	Common.CheckErr(err)
	defer f.Close()
	Common.CheckErr(ttTemplate.ExecuteTemplate(f, config.Workspace.TemplateStruct, structTemplate))
}

// 处理mapper
func MapperHandler(curTableName string) {
	TableNameCamel := HandlingStringsBig(curTableName)
	TableNameCamel = fmt.Sprintf("%s%s%s", config.Workspace.ClassNamePrefix, TableNameCamel, config.Workspace.ClassNameSuffix)
	mapperTemplate.PackageStruct = config.CurStructPackage + "." + TableNameCamel
	mapperTemplate.InterfaceName = TableNameCamel + "Mapper"
	mapperTemplate.StructName = TableNameCamel

	//创建目录
	Common.CheckErr(Common.MkdirAll(config.CurMapperPath))
	//写文件
	tempPath := fmt.Sprintf("./%s/%s.java", config.CurMapperPath, mapperTemplate.InterfaceName)
	if !config.Workspace.ForceCover && Common.FileExists(tempPath) {
		//不覆盖
		tempPath = fmt.Sprintf("./%s/%s-copy.java", config.CurMapperPath, mapperTemplate.InterfaceName)
	}
	f, err := os.Create(tempPath)
	Common.CheckErr(err)
	defer f.Close()
	Common.CheckErr(ttTemplate.ExecuteTemplate(f, config.Workspace.TemplateMapper, mapperTemplate))
}

// 处理Mybatis
func MybatisHandler(curDBName string, curTableName string, sFieldInfos []Model.SFieldInfo, size int) {
	var sFieldInfo Model.SFieldInfo
	var CustomResultMapBuffer bytes.Buffer
	var TableColumnsBuffer bytes.Buffer
	var EntityPropertiesBuffer bytes.Buffer
	var BatchEntityPropertiesBuffer bytes.Buffer
	var UpdateContentBuffer bytes.Buffer
	var LimitContentBuffer bytes.Buffer
	strLimit := `            <!-- 
            <if test="id != null">
                AND id=#{id}
            </if>
            <if test="idList != null">
                AND id IN
                <foreach collection="idList" item="it" separator="," open="(" close=")">
                    #{it}
                </foreach>
            </if>
            -->
`
	LimitContentBuffer.WriteString(strLimit)
	for i := 0; i < size; i++ {
		sFieldInfo = sFieldInfos[i]
		CustomResultMapBuffer.WriteString(fmt.Sprintf("        <result property=\"%s\" column=\"%s\"/>\r\n", sFieldInfo.FieldNameCamel, sFieldInfo.FieldName))
		TableColumnsBuffer.WriteString(fmt.Sprintf("        %s,\r\n", sFieldInfo.FieldName))
		EntityPropertiesBuffer.WriteString(fmt.Sprintf("        #{%s},\r\n", sFieldInfo.FieldNameCamel))
		BatchEntityPropertiesBuffer.WriteString(fmt.Sprintf("        #{item.%s},\r\n", sFieldInfo.FieldNameCamel))

		if "char" == sFieldInfo.FieldType ||
			"varchar" == sFieldInfo.FieldType ||
			"tinytext" == sFieldInfo.FieldType ||
			"text" == sFieldInfo.FieldType ||
			"mediumtext" == sFieldInfo.FieldType ||
			"longtext" == sFieldInfo.FieldType {
			//字符串类型
			UpdateContentBuffer.WriteString(fmt.Sprintf("            <if test=\"%s != null and %s != ''\">%s = #{%s},</if>\r\n", sFieldInfo.FieldNameCamel, sFieldInfo.FieldNameCamel, sFieldInfo.FieldName, sFieldInfo.FieldNameCamel))
			LimitContentBuffer.WriteString(fmt.Sprintf("            <if test=\"%s != null and %s != ''\">AND %s LIKE CONCAT(CONCAT('%%',#{%s},'%%'))</if>\r\n", sFieldInfo.FieldNameCamel, sFieldInfo.FieldNameCamel, sFieldInfo.FieldName, sFieldInfo.FieldNameCamel))
		} else {
			//非字符串
			UpdateContentBuffer.WriteString(fmt.Sprintf("            <if test=\"%s != null\">%s = #{%s},</if>\r\n", sFieldInfo.FieldNameCamel, sFieldInfo.FieldName, sFieldInfo.FieldNameCamel))
			LimitContentBuffer.WriteString(fmt.Sprintf("            <if test=\"%s != null\">AND %s = #{%s}</if>\r\n", sFieldInfo.FieldNameCamel, sFieldInfo.FieldName, sFieldInfo.FieldNameCamel))
		}

	}
	TableNameCamel := HandlingStringsBig(curTableName)
	TableNameCamel = fmt.Sprintf("%s%s%s", config.Workspace.ClassNamePrefix, TableNameCamel, config.Workspace.ClassNameSuffix)
	if config.MySql.IsDBNameInMyBatis {
		mybatisTemplate.DBName = fmt.Sprintf("`%s`.", curDBName)
	}
	mybatisTemplate.TableName = curTableName
	mybatisTemplate.MapperPath = config.CurMapperPackage + "." + TableNameCamel + "Mapper"
	mybatisTemplate.StructPath = config.CurStructPackage + "." + TableNameCamel
	mybatisTemplate.CustomResultMap = CustomResultMapBuffer.String()
	mybatisTemplate.CustomResultMap = mybatisTemplate.CustomResultMap[:len(mybatisTemplate.CustomResultMap)-2]
	//处理一下 START
	mybatisTemplate.TableColumns = TableColumnsBuffer.String()
	mybatisTemplate.TableColumns = mybatisTemplate.TableColumns[:len(mybatisTemplate.TableColumns)-3]

	mybatisTemplate.EntityProperties = EntityPropertiesBuffer.String()
	mybatisTemplate.EntityProperties = mybatisTemplate.EntityProperties[:len(mybatisTemplate.EntityProperties)-3]

	mybatisTemplate.BatchEntityProperties = BatchEntityPropertiesBuffer.String()
	mybatisTemplate.BatchEntityProperties = mybatisTemplate.BatchEntityProperties[:len(mybatisTemplate.BatchEntityProperties)-3]
	//处理一下 END

	mybatisTemplate.UpdateContent = UpdateContentBuffer.String()
	mybatisTemplate.UpdateContent = mybatisTemplate.UpdateContent[:len(mybatisTemplate.UpdateContent)-2]
	mybatisTemplate.LimitContent = LimitContentBuffer.String()
	mybatisTemplate.LimitContent = mybatisTemplate.LimitContent[:len(mybatisTemplate.LimitContent)-2]

	//创建目录
	Common.CheckErr(Common.MkdirAll(config.CurXmlPath))
	//写文件
	tempPath := fmt.Sprintf("./%s/%s.xml", config.CurXmlPath, TableNameCamel+"Mapper")
	if !config.Workspace.ForceCover && Common.FileExists(tempPath) {
		//不覆盖
		tempPath = fmt.Sprintf("./%s/%s-copy.xml", config.CurXmlPath, TableNameCamel+"Mapper")
	}
	f, err := os.Create(tempPath)
	Common.CheckErr(err)
	defer f.Close()
	Common.CheckErr(ttTemplate.ExecuteTemplate(f, config.Workspace.TemplateMybatis, mybatisTemplate))
}

// 处理表
func TableProcessing(curDBName string, curTableName string) {
	if curDBName == "" || curTableName == "" {
		return
	}

	strSql := fmt.Sprintf("SELECT `COLUMN_NAME`, `DATA_TYPE`, `COLUMN_COMMENT` from information_schema.columns WHERE table_schema = '%s' AND table_name = '%s' ORDER BY ordinal_position", curDBName, curTableName)
	rows, err := db.Query(strSql)
	Common.CheckErr(err)
	defer rows.Close()

	var sFieldInfos []Model.SFieldInfo

	var fieldName sql.NullString
	var fieldType sql.NullString
	var fieldComment sql.NullString
	for rows.Next() {
		_, err = rows.Columns()
		Common.CheckErr(err)
		err = rows.Scan(
			&fieldName,
			&fieldType,
			&fieldComment,
		)

		if !fieldName.Valid || !fieldType.Valid {
			continue
		}
		fieldName.String = strings.ToLower(fieldName.String)
		fieldType.String = strings.ToLower(fieldType.String)

		var sFieldInfo Model.SFieldInfo
		sFieldInfo.FieldName = fieldName.String
		sFieldInfo.FieldNameCamel = HandlingStringsLittle(fieldName.String)
		sFieldInfo.FieldType = fieldType.String
		sFieldInfo.FieldComment = fieldComment.String
		sFieldInfos = append(sFieldInfos, sFieldInfo)
	}

	if len(sFieldInfos) <= 0 {
		Common.CheckErr(errors.New("ERROR!!! len(sFieldInfos) <= 0"))
		return
	}

	StructHandler(curTableName, sFieldInfos, len(sFieldInfos))
	MapperHandler(curTableName)
	MybatisHandler(curDBName, curTableName, sFieldInfos, len(sFieldInfos))
}

func dealWithPath(path string) string {
	if "/" == path[len(path)-1:len(path)] {
		return path[:len(path)-1]
	} else {
		return path
	}
}

func main() {
	viper.SetConfigName("AutoJavaCode")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./")
	Common.CheckErr(viper.ReadInConfig())
	Common.CheckErr(viper.Unmarshal(&config))

	isFind := false
	for i := 0; i < len(config.DirStruct); i++ {
		for j := 0; j < len(config.DirStruct[i].Name); j++ {
			if config.DirStruct[i].Name[j] == config.Workspace.DirName {
				config.CurStructPath = dealWithPath(config.DirStruct[i].StructPath)
				config.CurStructPackage = config.DirStruct[i].StructPackage
				config.CurMapperPath = dealWithPath(config.DirStruct[i].MapperPath)
				config.CurMapperPackage = config.DirStruct[i].MapperPackage
				config.CurXmlPath = dealWithPath(config.DirStruct[i].XmlPath)
				isFind = true
			}
		}
	}
	if !isFind {
		fmt.Println("UnConfigured DBName")
		return
	}

	//bitConfig, _ := json.Marshal(config)
	//fmt.Println(string(bitConfig))

	//fmt.Println(config.CurStructPath)
	//fmt.Println(config.CurStructPackage)
	//fmt.Println(config.CurMapperPath)
	//fmt.Println(config.CurMapperPackage)
	//fmt.Println(config.CurXmlPath)

	//初始化模板
	initTemplate()
	//初始化Mysql
	Common.CheckErr(initDB(&config))
	defer db.Close()

	//处理数据库
	for i := 0; i < len(config.Workspace.TableName); i++ {
		TableProcessing(config.Workspace.DBName, config.Workspace.TableName[i])
	}

	fmt.Println("Completed!!!")
}

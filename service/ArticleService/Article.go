package ArticleService

import (
	"errors"
	"mime/multipart"
	"note-gin/models"
	"note-gin/pkg/HttpCode"
	"note-gin/pkg/RedisClient"
	"note-gin/pkg/utils"
	"note-gin/service/FolderService"
	"note-gin/view/ArticleView"
	"note-gin/view/common"
	"strings"
	"time"
)

func ArticleDownLoad(ID string) (string, string) {
	article := GetArticleDetail(ID)
	return article.Title, article.MkValue
}

func GetArticleByPage(page int) ([]ArticleView.ArticleInfo, int) {
	articles := models.Article{}.GetMany(page)
	total := models.Article{}.Count()
	ArticleInfos := ArticleView.ToArticleInfos(articles)
	return ArticleInfos, total
}

func GetArticleDetail(ID string) ArticleView.ArticleDetail {
	article := models.Article{}
	article.ID = int64(utils.StrToInt(ID))
	article.GetArticleInfo()
	articleDetail := ArticleView.ToArticleDetail(article)
	return articleDetail
}

func ClearRubbish() {
	models.Article{}.ClearRubbish()
}

func Delete(ID string) int64 {
	article := models.Article{}
	article.ID = int64(utils.StrToInt(ID))
	article.Delete()
	return article.ID
}

func DeleteMany(IDs []string) {
	models.Article{}.DeleteMany(IDs)
}

func GetRubbishArticles() common.DataList {
	articles := models.Article{}.GetDeletedArticle()
	respDataList := common.DataList{
		Items: articles,
		Total: int64(len(articles)),
	}
	return respDataList
}

func ArticleRecover(ID string) error {
	article := models.Article{}
	article.ID = int64(utils.StrToInt(ID))
	return article.Recover()
}

func Add(articleEditView *ArticleView.ArticleEditView) {
	article := models.Article{}
	article.Title = articleEditView.Title
	if articleEditView.FolderTitle != "Home" {
		article.FolderID = FolderService.GetFolderByTitle(articleEditView.FolderTitle).ID
	}
	article.Add() //这里调用的方法必须是指针类型
	articleEditView.FolderID = article.FolderID
	articleEditView.DirPath = append(articleEditView.DirPath, articleEditView.FolderID) //先添加自己的根目录
	models.Folder{}.GetFolderPath(articleEditView.FolderID, &articleEditView.DirPath)   //查找路径
}

func Update(articleEditView *ArticleView.ArticleEditView) {
	article := models.Article{}
	article.ID = articleEditView.ID
	article.UpdatedAt = time.Now()
	if len(articleEditView.DirPath) != 0 {
		article.FolderID = articleEditView.DirPath[len(articleEditView.DirPath)-1]
	}

	article.MkValue = articleEditView.MkValue
	article.Title = articleEditView.Title
	article.Update()

	articleEditView.UpdatedAt = article.UpdatedAt.Format("2006-01-02")
	articleEditView.CreatedAt = article.UpdatedAt.Format("2006-01-02")
	articleEditView.ID = article.ID
}

func Edit(articleEditView *ArticleView.ArticleEditView) {
	//目录路径回溯
	articleEditView.DirPath = append(articleEditView.DirPath, articleEditView.FolderID) //先添加自己的根目录
	FolderService.GetFolderPath(articleEditView.FolderID, &articleEditView.DirPath)     //查找路径
}

func SetTag(articleInfo ArticleView.ArticleInfo) {
	article := ArticleView.ToArticle(articleInfo)
	article.SetTag()
}

func TempArticleEditGet() (ArticleView.ArticleEditView, bool) {
	articleEditView := ArticleView.ArticleEditView{}
	ok := RedisClient.GetTempEdit(&articleEditView)
	return articleEditView, ok
}

func TempArticleEditDelete() int64 {
	return RedisClient.DeleteTempEdit()
}

func TempArticleEditSave(articleEditView ArticleView.ArticleEditView) bool {
	flag := RedisClient.SaveTempEdit(articleEditView)
	if strings.ToLower(flag) == "ok" {
		return true
	} else {
		return false
	}
}

func UploadArticle(files map[string][]*multipart.FileHeader, folder_title string, file_name *string) (bool, error) {
	folder_id := FolderService.GetFolderByTitle(folder_title).ID
	for name, file := range files {
		names := strings.Split(name, ".")
		typeName := names[1]
		if typeName != "md" {
			return false, errors.New(HttpCode.HttpMsg[HttpCode.ERROR_FILE_TYPE])
		}

		fp, _ := file[0].Open()
		b := make([]byte, file[0].Size)
		fp.Read(b)

		article := models.Article{}
		article.Title = names[0]
		*file_name = article.Title
		isExist := article.IsExist()
		if isExist != true {
			article.FolderID = folder_id
			article.MkValue = string(b)
			article.Add()
			return true, nil
		} else { //存在同名文件则更新 不管是否是在同一个目录下  【整个系统不允许出现同名文件】
			article.GetArticleInfoByTitle()
			article.FolderID = folder_id
			article.MkValue = string(b)
			article.Update()
			return false, errors.New(HttpCode.HttpMsg[HttpCode.FILE_IS_EXIST_AND_UPDATE])
		}

	}
	return false, nil
}

package constants

// Shared user-facing and backend messages. A wording change here may affect
// frontend hints, backend responses and log content simultaneously.
const (
	MsgRegisterOK           = "注册成功"
	MsgLoginOK              = "登录成功"
	MsgLogoutOK             = "退出登录成功"
	MsgPlantAddedToGarden   = "已加入我的花园"
	MsgPlantRemovedGarden   = "已从我的花园移除"
	MsgFavoriteAdded        = "收藏成功"
	MsgFavoriteRemoved      = "已取消收藏"
	MsgArticlePublished     = "文章发布成功"
	MsgArticleSaved         = "文章已保存"
	MsgReminderCreated         = "养护提醒已创建"
	MsgReminderDone            = "提醒已标记完成"
	MsgReminderAlreadyDone     = "该提醒已处理，下一期已生成，请勿重复完成"
	MsgReminderUpdated         = "提醒排期已更新，下一期已重新计算"
	MsgReminderReopened        = "提醒已恢复为待处理"
	MsgReminderAwaiting        = "植物已移出，未完成提醒待确认"
	MsgReminderCanceled        = "待确认提醒已取消"
	MsgReminderTransferred     = "提醒已转移给同品种的另一盆"
	MsgQuestionCreated      = "问题发布成功"
	MsgAnswerAdopted        = "已采纳该回答"
	MsgAnswerLiked          = "点赞成功"
	MsgProfileUpdated       = "资料已更新"
	MsgUploadOK             = "上传成功"
	MsgInvalidCredentials   = "用户名或密码错误"
	MsgUsernameTaken        = "用户名已存在"
	MsgEmailTaken           = "邮箱已被注册"
)

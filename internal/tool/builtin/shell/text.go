package shell

const (
	DescShell = "在项目工作区运行 shell 命令。background=true 可后台启动，用 job_id 轮询或取消。"

	SchemaShellDescription = "用一句自然语言简述本次 shell 调用的目的，不要重复完整命令"
	SchemaShellCommand     = "Shell 命令（同步/后台启动时必填）"
	SchemaBackground       = "为 true 时在后台启动并返回 job_id"
	SchemaJobID            = "轮询后台任务输出/状态；cancel=true 时取消任务"
	SchemaCancel           = "终止后台任务（需 job_id）"
	SchemaListJobs         = "列出本项目的后台任务"

	ErrBackgroundUnavailable = "shell 后台任务不可用"
	ErrCommandRequired       = "command 为必填项（或使用 job_id / list_jobs）"
	ErrCommandRequiredSync   = "command 为必填项"

	ResultNoBackgroundJobs = "无后台 shell 任务。"
	ResultNoOutput           = "（无输出）"
	ResultKilledJob          = "已终止后台任务 %s（%s）"
	ResultBackgroundStarted  = "后台任务已启动\njob_id: %s\npid: %d\nstatus: %s\ncommand: %s"
	ResultJobHeader          = "job_id: %s\nstatus: %s\ncommand: %s\npid: %d\nstarted: %s\n"
	ResultJobFinished        = "finished: %s\n"
	ResultJobExitCode        = "exit_code: %d\n"
	ResultStdout             = "\nstdout:\n"
	ResultStderr             = "\nstderr:\n"
	ResultNoOutputYet        = "\n（尚无输出）\n"
	ResultJobListHeader      = "后台 shell 任务：\n"
	ResultJobListLine        = "  %s  %s  pid=%d%s  %q\n"
	ResultExitPrefix         = "exit: "
)

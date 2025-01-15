package main

import (
	// Make sure to import swag docs, don't remove
	_ "fireboom-server/app/docs"
	"fireboom-server/cmd"
	"os"
)

var (
	FbVersion string
	FbCommit  string
)

func main() {
	// _ = os.Chdir("/path/to/your/project")
	// _ = os.Chdir("/Users/xufeixiang/IdeaProjects/backend_face_recognition_boarding")
	// _ = os.Chdir("/Users/xufeixiang/IdeaProjects/case-freetalk")
	// _ = os.Chdir("/Users/xufeixiang/IdeaProjects/amis-admin/backend")
	_ = os.Chdir("/Users/xufeixiang/IdeaProjects/turing_ant/turing_ant_backend")
	// _ = os.Chdir("/Users/xufeixiang/IdeaProjects/dajiache-backend/backend")
	// _ = os.Chdir("/Users/xufeixiang/IdeaProjects/dianshang_5l")
	// _ = os.Chdir("/Users/xufeixiang/IdeaProjects/init-todo")
	cmd.Execute(FbVersion, FbCommit)
}

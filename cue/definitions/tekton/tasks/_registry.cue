package tasks

#TaskRegistry: {
	"git-clone":           #GitClone
	"kaniko-build":        #KanikoBuild
	"git-update-manifest": #GitUpdateManifest
}

#RenderTask: {
	taskName: string
	output:   #TaskRegistry[taskName]
}

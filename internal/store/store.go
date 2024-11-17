package store

type Persistence struct {
	Deployment DeploymentStorer
	Namespace  NamespaceStorer
}

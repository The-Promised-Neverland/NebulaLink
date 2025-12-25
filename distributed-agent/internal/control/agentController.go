package control

type AgentController interface {
	Restart() error
	Uninstall() error
}

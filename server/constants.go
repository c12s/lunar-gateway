package server

const (
	BLACKHOLE = "blackhole"
	CELESTIAL = "celestial"
	APOLLO    = "apollo"
	STELLAR   = "stellar"
	MERIDIAN  = "meridian"
	MAGNETAR  = "magnetar"

	all = "all"
	any = "any"

	file   = "file"
	env    = "env"
	action = "action"

	at_once        = "AtOnce"
	rolling_update = "RollingUpdate"
	canary         = "Canary"

	compare = "compare"
	labels  = "labels"
	sep     = ":"
	kind    = "kind"

	top  = "top"
	from = "from"
	to   = "to"

	user       = "user"
	ns_key     = "namespace"
	labels_key = "labels"

	Configs    = "Configs"
	Secrets    = "Secrets"
	Actions    = "Actions"
	Namespaces = "Namespaces"
	Roles      = "Roles"
	Topology   = "Topology"
)

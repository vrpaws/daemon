package api

type Server interface {
	ValidUser(string) error // integrate with flight.Cache to prevent api spam
}

package vrcx

type Screenshot struct {
	Application string `json:"application"`
	Version     int64  `json:"version"`
	Author      User   `json:"author"`
	World       World  `json:"world"`
	Players     []User `json:"players"`
}

type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type World struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	InstanceID string `json:"instanceId"`
}

package vrpaws

type Me struct {
	User User `json:"user"`
}

type User struct {
	CreationTime    float64 `json:"_creationTime"`
	ID              string  `json:"_id"`
	AccessToken     string  `json:"accessToken"`
	IsProfilePublic bool    `json:"isProfilePublic"`
	UserID          string  `json:"userId"`
	Username        string  `json:"username"`
}

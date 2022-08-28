package main

import "context"

type ArenaUpdate struct {
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"_links"`
	Arena struct {
		Dimensions []int                  `json:"dims"`
		State      map[string]PlayerState `json:"state"`
	} `json:"arena"`
}

type PlayerState struct {
	URL       string `json:"-"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
	WasHit    bool   `json:"wasHit"`
	Score     int    `json:"score"`
}

func (p PlayerState) GetDirection() Direction {
	switch p.Direction {
	case "N":
		return North
	case "W":
		return West
	case "E":
		return East
	default:
		return South
	}
}

type Game struct {
	Arena            Arena
	PlayerStateByURL map[string]PlayerState
	LeaderBoard      LeaderBoard
}

const (
	defaultAttackRange int = 3
)

func NewGame() Game {
	return Game{
	}
}

func (g *Game) UpdateArena(ctx context.Context, a ArenaUpdate) {
	ctx, span := tracer.Start(ctx, "Game.UpdateArena")
	defer span.End()

	width := a.Arena.Dimensions[0]
	height := a.Arena.Dimensions[1]
	arena := NewArena(width, height)

	g.LeaderBoard = nil
	for k, v := range a.Arena.State {
		v.URL = k
		arena.PutPlayer(v)
		g.LeaderBoard = append(g.LeaderBoard, v)
	}

	g.Arena = arena
	g.PlayerStateByURL = a.Arena.State
	g.UpdateLeaderBoard(ctx)
}

func (g Game) Player(url string) *Player {
	pState := g.PlayerStateByURL[url]
	player := NewPlayerWithUrl(url, pState)
	player.Game = g
	return player
}

func (g Game) Update(player *Player) {
	updatedPlayer := g.PlayerStateByURL[player.Name]
	player.X = updatedPlayer.X
	player.Y = updatedPlayer.Y
	player.Direction = updatedPlayer.Direction
	player.WasHit = updatedPlayer.WasHit
	player.Score = updatedPlayer.Score
	player.Game = g
}

func (g *Game) UpdateLeaderBoard(ctx context.Context) {
	ctx, span := tracer.Start(ctx, "Game.UpdateLeaderBoard")
	defer span.End()

	g.LeaderBoard.Sort()
}

func (g Game) GetPlayerStateByPosition(p Point) (PlayerState, bool) {
	player := g.Arena.Grid[p.Y][p.X].Player
	if player == nil {
		return PlayerState{}, false
	}
	return *player, true
}

// GetPlayerByRank rank starts from 0 (highest rank)
func (g Game) GetPlayerByRank(rank int) *Player {
	ps := g.LeaderBoard.GetPlayerByRank(rank)
	if ps == nil {
		return nil
	}
	return g.GetPlayerByPosition(Point{ps.X, ps.Y})
}

func (g Game) GetPlayerByPosition(p Point) *Player {
	pState := g.Arena.Grid[p.Y][p.X].Player
	if pState == nil {
		return nil
	}
	player := NewPlayerWithUrl(pState.URL, *pState)
	player.Game = g
	return player
}

// ObstacleMap return map which denotes whether a cell adalah obstacle or not
func (g Game) ObstacleMap(ctx context.Context) [][]bool {
	ctx, span := tracer.Start(ctx, "Game.ObstacleMap")
	defer span.End()

	m := make([][]bool, g.Arena.Height)
	for i, _ := range m {
		row := make([]bool, g.Arena.Width)
		m[i] = row
	}

	for _, ps := range g.PlayerStateByURL {
		if !ps.WasHit {
			m[ps.Y][ps.X] = true
			continue
		}

		player := g.GetPlayerByPosition(Point{ps.X, ps.Y})
		if player == nil {
			// TODO warning
			continue
		}

		// TODO seems like redundant code
		left := player.FindShooterOnDirection(ctx, player.GetDirection().Left())
		for _, l := range left {
			npt := l.GetPosition()
			for ctr := 0; ctr < defaultAttackRange; ctr++ {
				npt = npt.TranslateToDirection(1, l.GetDirection())
				if !g.Arena.IsValid(npt) {
					break
				}
				m[npt.Y][npt.X] = m[npt.Y][npt.X] || l.CanHitPoint(ctx, npt)
			}
		}

		front := player.FindShooterOnDirection(ctx, player.GetDirection())
		for _, l := range front {
			npt := l.GetPosition()
			for ctr := 0; ctr < defaultAttackRange; ctr++ {
				npt = npt.TranslateToDirection(1, l.GetDirection())
				if !g.Arena.IsValid(npt) {
					break
				}
				m[npt.Y][npt.X] = m[npt.Y][npt.X] || l.CanHitPoint(ctx, npt)
			}
		}

		back := player.FindShooterOnDirection(ctx, player.GetDirection().Backward())
		for _, l := range back {
			npt := l.GetPosition()
			for ctr := 0; ctr < defaultAttackRange; ctr++ {
				npt = npt.TranslateToDirection(1, l.GetDirection())
				if !g.Arena.IsValid(npt) {
					break
				}
				m[npt.Y][npt.X] = m[npt.Y][npt.X] || l.CanHitPoint(ctx, npt)
			}
		}

		right := player.FindShooterOnDirection(ctx, player.GetDirection().Right())
		for _, l := range right {
			npt := l.GetPosition()
			for ctr := 0; ctr < defaultAttackRange; ctr++ {
				npt = npt.TranslateToDirection(1, l.GetDirection())
				if !g.Arena.IsValid(npt) {
					break
				}
				m[npt.Y][npt.X] = m[npt.Y][npt.X] || l.CanHitPoint(ctx, npt)
			}
		}
	}

	return m
}
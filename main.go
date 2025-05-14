package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Card struct {
	Suit  string
	Value int
	Type  string // "Monster", "Weapon", "Potion"
}

type model struct {
	health         int
	dungeon        []Card
	room           []Card
	equippedWeapon Card
	discardPile    []Card
	selectedCard   int // Index of the selected card in the room
	cardsChosen    int
	weaponLimit    int // The maximum value a weapon can be used on
}

func initialModel() *model {
	// Initialize the random number generator
	rand.Seed(time.Now().UnixNano())

	// Create the deck
	deck := createDeck()

	// Shuffle the deck
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})

	m := &model{
		health:         20,
		dungeon:        deck,
		room:           []Card{},
		equippedWeapon: Card{}, // Empty card
		discardPile:    []Card{},
		selectedCard:   -1, // -1 means no card is selected
		cardsChosen:    0,
		weaponLimit:    14, // Can use weapon on any monster to start
	}

	// Deal initial room
	m.dealRoom()
	return m
}

func createDeck() []Card {
	deck := []Card{}

	// Add Clubs and Spades (Monsters)
	for suit := range []string{"Club", "Spade"} {
		for i := 2; i <= 14; i++ {
			card := Card{
				Suit:  []string{"Club", "Spade"}[suit],
				Value: i,
				Type:  "Monster",
			}
			deck = append(deck, card)
		}
	}

	// Add Diamonds (Weapons)
	for i := 2; i <= 10; i++ {
		card := Card{
			Suit:  "Diamond",
			Value: i,
			Type:  "Weapon",
		}
		deck = append(deck, card)
	}

	// Add Hearts (Potions)
	for i := 2; i <= 10; i++ {
		card := Card{
			Suit:  "Heart",
			Value: i,
			Type:  "Potion",
		}
		deck = append(deck, card)
	}

	return deck
}

func (m *model) dealRoom() {
	// Clear the room
	m.room = []Card{}

	// Deal 4 cards from the dungeon to the room
	for i := 0; i < 4; i++ {
		if len(m.dungeon) > 0 {
			card := m.dungeon[0]
			m.dungeon = m.dungeon[1:]
			m.room = append(m.room, card)
		} else {
			// Dungeon is empty, handle this case (e.g., reshuffle discard pile)
			fmt.Println("Dungeon is empty!") // For now, just print a message
			break                               // Stop dealing if the dungeon is empty
		}
	}
	m.cardsChosen = 0
	m.selectedCard = -1
}

func (m *model) discard(card Card) {
	m.discardPile = append(m.discardPile, card)
}

func (m *model) equipWeapon(card Card) {
	if (m.equippedWeapon != Card{}) {
		m.discard(m.equippedWeapon)
	}
	m.equippedWeapon = card
}

func (m *model) usePotion(card Card) {
	m.health += card.Value
	if m.health > 20 {
		m.health = 20
	}
	m.discard(card)
}

func (m *model) fightMonster(card Card) {
	if (m.equippedWeapon == Card{}) {
		m.health -= card.Value
	} else {
		damage := card.Value - m.equippedWeapon.Value
		if damage > 0 {
			m.health -= damage
		}
		m.weaponLimit = card.Value
	}
	m.discard(card)
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "d":
			m.dealRoom()
			return m, nil
		case "1":
			return m.selectCard(0), nil
		case "2":
			return m.selectCard(1), nil
		case "3":
			return m.selectCard(2), nil
		case "4":
			return m.selectCard(3), nil
		// Handle game over and restart
		case "r":
			if m.health <= 0 {
				return initialModel(), nil // Restart the game
			}
		}
	}
	return m, nil
}

func (m *model) selectCard(index int) *model {
	if index >= 0 && index < len(m.room) && m.cardsChosen < 3 {
		card := m.room[index]
		m.selectedCard = index
		m.cardsChosen++

		// Take action based on card type
		switch card.Type {
		case "Weapon":
			m.equipWeapon(card)
		case "Potion":
			m.usePotion(card)
		case "Monster":
			m.fightMonster(card)
		}

		// Remove the card from the room
		m.room = append(m.room[:index], m.room[index+1:]...)

		//If 3 cards have been chosen, discard the remaining card and deal a new room
		if m.cardsChosen == 3 {
			if len(m.room) > 0 {
				m.discard(m.room[0])
				m.room = []Card{}
			}
			m.dealRoom()
		}

	} else {
		m.selectedCard = -1
		fmt.Println("Invalid card selection")
	}
	return m
}

func (m *model) View() string {
	s := "--------------------------------------------------\n"
	if m.health <= 0 {
		s += "|             Game Over!             |\n"
		s += "| Press 'r' to restart the game.   |\n"
		s += "--------------------------------------------------\n"
	} else {
		s += fmt.Sprintf("| Health: %-31d |\n", m.health)
		s += "--------------------------------------------------\n"
		s += fmt.Sprintf("| Dungeon: %-27d Cards |\n", len(m.dungeon))
		s += "--------------------------------------------------\n"
		roomStr := ""
		for i, card := range m.room {
			selected := ""
			if i == m.selectedCard {
				selected = "*" // Mark the selected card
			}
			roomStr += fmt.Sprintf("[%d:%s%s %d]", i+1, selected, card.Suit, card.Value)
		}
		s += fmt.Sprintf("| Room: %-34s |\n", roomStr)
		s += "--------------------------------------------------\n"
		s += fmt.Sprintf("| Equipped Weapon: %-10s %-9d |\n", m.equippedWeapon.Suit, m.equippedWeapon.Value)
		s += "--------------------------------------------------\n"
		s += fmt.Sprintf("| Discard Pile: %-23d |\n", len(m.discardPile))
		s += "--------------------------------------------------\n"
	}
	return s
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

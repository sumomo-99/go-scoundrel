package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var debugMode = false // Enable or disable debug mode

type Card struct {
	Suit      string
	Value     int
	Type      string // "Monster", "Weapon", "Potion"
	MonsterValue int // Value of the monster slain by this weapon
}

type model struct {
	health         int
	dungeon        []Card
	room           []Card
	equippedWeapon Card
	discardPile    []Card
	selectedCard   int // Index of the selected card in the room
	cardsChosen    int
	weaponLimit    int           // The maximum value a weapon can be used on
	choosingFight  bool          // True if the player is choosing how to fight
	fightingBarehanded bool // True if the player chose to fight barehanded
	avoidedLastRoom bool          // True if the player avoided the room last turn
	potionUsedThisTurn bool
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
		selectedCard:   -1,      // -1 means no card is selected
		cardsChosen:    0,
		weaponLimit:    14,      // Can use weapon on any monster to start
		choosingFight:  false, // Player is not choosing how to fight
		fightingBarehanded: false,
		avoidedLastRoom: false,
		potionUsedThisTurn: false,
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

func (m *model) calculateScore() int {
	if m.health <= 0 {
		score := m.health // Negative score if life reaches 0
		// Find all remaining monsters in the Dungeon and subtract their values
		for _, card := range m.dungeon {
			if card.Type == "Monster" {
				score -= card.Value
			}
		}
		return score
	}

	// If you have made your way through the entire dungeon, your score is your positive life
	score := m.health
	// If your life is 20, and your last card was a health potion, your life + the value of that potion.
	if m.health == 20 && len(m.discardPile) > 0 && m.discardPile[len(m.discardPile)-1].Type == "Potion" {
		score += m.discardPile[len(m.discardPile)-1].Value
	}
	return score
}

func (m *model) dealRoom() {
	m.avoidedLastRoom = false // Reset avoidedLastRoom at the start of the turn
	m.potionUsedThisTurn = false

	// Deal cards from the dungeon to the room until there are 4 cards
	for len(m.room) < 4 {
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
	m.equippedWeapon.MonsterValue = 0 // Reset monster value when equipping a new weapon
}

func (m *model) usePotion(card Card) {
	if !m.potionUsedThisTurn {
		m.health += card.Value
		if m.health > 20 {
			m.health = 20
		}
		m.discard(card)
		m.potionUsedThisTurn = true

		// Remove the card from the room
		m.room = append(m.room[:index], m.room[index+1:]...)

		// If 3 cards have been chosen (or removed), deal a new room
		if 4-len(m.room) == 3 {
			m.dealRoom()
		}

	} else {
		m.discard(card) // Discard the potion without using it
	}
}

func (m *model) fightMonster(card Card) {
	m.choosingFight = true
}

func (m *model) finishFight() (tea.Model, tea.Cmd) {
	// Check if the selected card is still valid
	if m.selectedCard < 0 || m.selectedCard >= len(m.room) {
		m.choosingFight = false
		return m, nil
	}

	card := m.room[m.selectedCard]

	if m.fightingBarehanded {
		m.health -= card.Value
	} else {
		if (m.equippedWeapon == Card{}) {
			m.health -= card.Value
		} else {
			// Check if the weapon can be used
			if card.Value > m.weaponLimit {
				m.health -= card.Value // Fight barehanded
			} else {
				damage := card.Value - m.equippedWeapon.Value
				if damage > 0 {
					m.health -= damage
				}
			}
		}
	}

	if !m.fightingBarehanded && (m.equippedWeapon != Card{}) && (card.Value <= m.weaponLimit) {
		m.weaponLimit = card.Value                  // Update weapon limit *after* the fight
		m.equippedWeapon.MonsterValue = card.Value // Store the monster's value on the weapon
	}
	m.discard(card)

	// Remove the card from the room
	m.room = append(m.room[:m.selectedCard], m.room[m.selectedCard+1:]...)
	m.selectedCard = -1
	m.choosingFight = false

	// If 3 cards have been chosen (or removed), deal a new room
	if 4-len(m.room) == 3 {
		m.dealRoom()
	}
	return m, nil
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
		case "a":
			// Avoid the room
			if !m.avoidedLastRoom {
				// Place the cards at the bottom of the dungeon
				for _, card := range m.room {
					m.dungeon = append(m.dungeon, card)
				}
				// Clear the room
				m.room = []Card{}
				m.dealRoom()
				m.avoidedLastRoom = true // Mark that the room was avoided
				return m, nil
			}
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

		// Handle choosing to fight barehanded or with a weapon
		case "b":
			if m.choosingFight {
				m.fightingBarehanded = true
				model, cmd := m.finishFight()
				return model, cmd
			}
		case "w":
			if m.choosingFight {
				m.fightingBarehanded = false
				model, cmd := m.finishFight()
				return model, cmd
			}

		// Handle game over and restart
		case "r":
			if m.health <= 0 {
				return initialModel(), nil // Restart the game
			}
		case "t":
			debugMode = !debugMode
			return m, nil
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
			m.room = append(m.room[:index], m.room[index+1:]...)
		case "Potion":
			m.usePotion(card)
			m.room = append(m.room[:index], m.room[index+1:]...)
		case "Monster":
			m.selectedCard = index
			m.fightMonster(card)
			m.choosingFight = true
		}

	} else {
		fmt.Println("Invalid card selection")
	}
	return m
}

func (m *model) View() string {
	s := "--------------------------------------------------\n"
	if m.health <= 0 {
		s += "             Game Over!             \n"
		s += fmt.Sprintf("             Score: %-4d           \n", m.calculateScore())
		s += " Press 'r' to restart the game.   \n"
		s += "--------------------------------------------------\n"
	} else {
		s += fmt.Sprintf(" Health ❤️: %-29d \n", m.health)
		s += "--------------------------------------------------\n"
		s += fmt.Sprintf(" Dungeon 💥: %-25d Cards \n", len(m.dungeon))

		// Debug mode: display room values
		if debugMode {
			s += " Debug: Room values:\n"
			for _, card := range m.room {
				s += fmt.Sprintf("   %v\n", card)
			}
			s += "--------------------------------------------------\n"
		} else {
			s += "--------------------------------------------------\n"
		}

		// Show avoid room option if not avoided last room
		if !m.avoidedLastRoom {
			s += " Avoid Room? (a)                      \n"
			s += "--------------------------------------------------\n"
		}

		roomStr := ""
		for i, card := range m.room {
			selected := ""
			if i == m.selectedCard {
				selected = "*" // Mark the selected card
			}
			roomStr += fmt.Sprintf("[%d:%s%s %d]", i+1, selected, card.Suit, card.Value)
		}

		s += fmt.Sprintf(" Room 🚪: %-32s \n", roomStr)
		s += "--------------------------------------------------\n"

		if m.choosingFight {
			s += " Fight Barehanded (b) or With Weapon (w)? \n"
			s += "--------------------------------------------------\n"
		} else {
			weaponStr := fmt.Sprintf("%s %d", m.equippedWeapon.Suit, m.equippedWeapon.Value)
			if m.equippedWeapon.MonsterValue > 0 {
				weaponStr += fmt.Sprintf(" (Monster: %d)", m.equippedWeapon.MonsterValue)
			}
			s += fmt.Sprintf(" Equipped Weapon 🗡️: %-28s \n", weaponStr)
			s += "--------------------------------------------------\n"
			s += fmt.Sprintf(" Discard Pile ♻️: %-21d \n", len(m.discardPile))
			s += "--------------------------------------------------\n"
		}
	}
	s += fmt.Sprintf(" Score 💰: %-30d \n", m.calculateScore())
	s += "--------------------------------------------------\n"
	return s
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

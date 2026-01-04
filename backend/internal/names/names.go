package names

import (
	"fmt"
	"math/rand"
)

// Famous singer first names - diverse genres and eras
var firstNames = []string{
	// Rock/Pop legends
	"Elvis", "Freddie", "David", "Mick", "John", "Paul", "Ringo", "George",
	"Stevie", "Prince", "Michael", "Whitney", "Aretha", "Tina", "Cher",
	"Madonna", "Cyndi", "Pat", "Bonnie", "Janis",
	// Modern pop
	"Beyoncé", "Adele", "Bruno", "Taylor", "Ariana", "Dua", "Billie",
	"Harry", "Shawn", "Camila", "Demi", "Selena", "Miley", "Katy",
	// R&B/Soul
	"Marvin", "Otis", "James", "Ray", "Sam", "Smokey", "Lionel",
	"Luther", "Usher", "Alicia", "Mary", "Lauryn", "Erykah",
	// Country
	"Dolly", "Johnny", "Willie", "Waylon", "Garth", "Shania", "Reba",
	"Carrie", "Blake", "Keith", "Faith", "Tim", "Kenny",
	// Hip-hop/Rap
	"Snoop", "Tupac", "Biggie", "Jay", "Kanye", "Drake", "Kendrick",
	"Nicki", "Cardi", "Missy", "Lauryn", "Queen",
	// Jazz/Blues
	"Ella", "Billie", "Nina", "Etta", "Bessie", "Sarah", "Dinah",
	"Louis", "Nat", "Frank", "Tony", "Dean", "Sammy",
	// International
	"Shakira", "Rihanna", "Celine", "Bjork", "Sade", "Enya",
	"Bono", "Sting", "Seal", "Enrique", "Julio",
	// Classic rock
	"Ozzy", "Axl", "Slash", "Eddie", "Kurt", "Chris", "Chester",
	"Robert", "Jimmy", "Roger", "Pete", "Keith", "Ronnie",
}

// Famous singer last names - mixed to create funny combinations
var lastNames = []string{
	// Rock legends
	"Presley", "Mercury", "Bowie", "Jagger", "Lennon", "McCartney",
	"Wonder", "Jackson", "Houston", "Franklin", "Turner", "Parton",
	// Modern stars
	"Swift", "Grande", "Mars", "Styles", "Eilish", "Lipa",
	"Lovato", "Gomez", "Cyrus", "Perry", "Gaga", "Beyoncé",
	// Soul/R&B
	"Gaye", "Redding", "Brown", "Charles", "Cooke", "Robinson",
	"Vandross", "Keys", "Blige", "Hill", "Badu",
	// Country
	"Cash", "Nelson", "Jennings", "Brooks", "Twain", "McEntire",
	"Underwood", "Shelton", "Urban", "Hill", "McGraw",
	// Hip-hop
	"Dogg", "Shakur", "Smalls", "Z", "West", "Lamar",
	"Minaj", "B", "Elliott", "Latifah",
	// Jazz/Blues
	"Fitzgerald", "Holiday", "Simone", "James", "Smith", "Vaughan",
	"Armstrong", "Cole", "Sinatra", "Bennett", "Martin", "Davis",
	// International
	"Dion", "Rihanna", "Shakira", "Twain", "Estefan",
	// Rock surnames
	"Osbourne", "Rose", "Van Halen", "Cobain", "Cornell", "Bennington",
	"Plant", "Page", "Waters", "Townshend", "Moon", "Wood",
	// Fun additions
	"Thundervoice", "Goldentone", "Silverpipe", "Velvetlungs",
	"Crescendo", "Falsetto", "Vibrato", "Octave", "Harmony",
}

// GenerateSingerName creates a random funny singer name
func GenerateSingerName() string {
	first := firstNames[rand.Intn(len(firstNames))]
	last := lastNames[rand.Intn(len(lastNames))]

	// 30% chance to add numbers at the end
	if rand.Intn(10) < 3 {
		num := rand.Intn(99) + 1
		return fmt.Sprintf("%s %s %d", first, last, num)
	}

	return fmt.Sprintf("%s %s", first, last)
}

// GenerateUniqueSingerName generates a name that's not in the existing names map
func GenerateUniqueSingerName(existingNames map[string]bool) string {
	maxAttempts := 100
	for i := 0; i < maxAttempts; i++ {
		name := GenerateSingerName()
		if !existingNames[name] {
			return name
		}
	}
	// Fallback: add random numbers to ensure uniqueness
	return fmt.Sprintf("%s %d", GenerateSingerName(), rand.Intn(9999))
}

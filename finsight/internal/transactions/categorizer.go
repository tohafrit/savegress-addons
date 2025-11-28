package transactions

import (
	"regexp"
	"strings"

	"github.com/savegress/finsight/pkg/models"
)

// Categorizer categorizes transactions based on merchant and description
type Categorizer struct {
	mccCategories     map[string]string
	merchantPatterns  map[string]*regexp.Regexp
	descriptionRules  []CategorizationRule
}

// CategorizationRule defines a rule for categorizing transactions
type CategorizationRule struct {
	Pattern  *regexp.Regexp
	Category string
	Priority int
}

// Category constants
const (
	CategoryGroceries      = "groceries"
	CategoryRestaurants    = "restaurants"
	CategoryTransportation = "transportation"
	CategoryUtilities      = "utilities"
	CategoryEntertainment  = "entertainment"
	CategoryShopping       = "shopping"
	CategoryHealthcare     = "healthcare"
	CategoryTravel         = "travel"
	CategoryFees           = "fees"
	CategoryTransfer       = "transfer"
	CategoryIncome         = "income"
	CategoryInvestment     = "investment"
	CategoryInsurance      = "insurance"
	CategoryEducation      = "education"
	CategorySubscription   = "subscription"
	CategoryOther          = "other"
)

// NewCategorizer creates a new transaction categorizer
func NewCategorizer() *Categorizer {
	c := &Categorizer{
		mccCategories:    make(map[string]string),
		merchantPatterns: make(map[string]*regexp.Regexp),
	}
	c.initializeMCCCategories()
	c.initializeMerchantPatterns()
	c.initializeDescriptionRules()
	return c
}

func (c *Categorizer) initializeMCCCategories() {
	// Grocery stores
	c.mccCategories["5411"] = CategoryGroceries
	c.mccCategories["5422"] = CategoryGroceries
	c.mccCategories["5441"] = CategoryGroceries
	c.mccCategories["5451"] = CategoryGroceries
	c.mccCategories["5462"] = CategoryGroceries

	// Restaurants & Food
	c.mccCategories["5812"] = CategoryRestaurants
	c.mccCategories["5813"] = CategoryRestaurants
	c.mccCategories["5814"] = CategoryRestaurants

	// Transportation
	c.mccCategories["4111"] = CategoryTransportation
	c.mccCategories["4112"] = CategoryTransportation
	c.mccCategories["4121"] = CategoryTransportation
	c.mccCategories["4131"] = CategoryTransportation
	c.mccCategories["5541"] = CategoryTransportation
	c.mccCategories["5542"] = CategoryTransportation

	// Utilities
	c.mccCategories["4814"] = CategoryUtilities
	c.mccCategories["4816"] = CategoryUtilities
	c.mccCategories["4899"] = CategoryUtilities
	c.mccCategories["4900"] = CategoryUtilities

	// Entertainment
	c.mccCategories["7832"] = CategoryEntertainment
	c.mccCategories["7841"] = CategoryEntertainment
	c.mccCategories["7911"] = CategoryEntertainment
	c.mccCategories["7922"] = CategoryEntertainment
	c.mccCategories["7929"] = CategoryEntertainment
	c.mccCategories["7932"] = CategoryEntertainment
	c.mccCategories["7933"] = CategoryEntertainment
	c.mccCategories["7941"] = CategoryEntertainment

	// Shopping
	c.mccCategories["5311"] = CategoryShopping
	c.mccCategories["5331"] = CategoryShopping
	c.mccCategories["5399"] = CategoryShopping
	c.mccCategories["5611"] = CategoryShopping
	c.mccCategories["5621"] = CategoryShopping
	c.mccCategories["5631"] = CategoryShopping
	c.mccCategories["5641"] = CategoryShopping
	c.mccCategories["5651"] = CategoryShopping
	c.mccCategories["5661"] = CategoryShopping
	c.mccCategories["5691"] = CategoryShopping
	c.mccCategories["5699"] = CategoryShopping

	// Healthcare
	c.mccCategories["5912"] = CategoryHealthcare
	c.mccCategories["8011"] = CategoryHealthcare
	c.mccCategories["8021"] = CategoryHealthcare
	c.mccCategories["8031"] = CategoryHealthcare
	c.mccCategories["8041"] = CategoryHealthcare
	c.mccCategories["8042"] = CategoryHealthcare
	c.mccCategories["8043"] = CategoryHealthcare
	c.mccCategories["8049"] = CategoryHealthcare
	c.mccCategories["8050"] = CategoryHealthcare
	c.mccCategories["8062"] = CategoryHealthcare
	c.mccCategories["8071"] = CategoryHealthcare
	c.mccCategories["8099"] = CategoryHealthcare

	// Travel
	c.mccCategories["3000"] = CategoryTravel
	c.mccCategories["3001"] = CategoryTravel
	c.mccCategories["4411"] = CategoryTravel
	c.mccCategories["4511"] = CategoryTravel
	c.mccCategories["4722"] = CategoryTravel
	c.mccCategories["7011"] = CategoryTravel
	c.mccCategories["7012"] = CategoryTravel

	// Insurance
	c.mccCategories["5960"] = CategoryInsurance
	c.mccCategories["6300"] = CategoryInsurance

	// Education
	c.mccCategories["8211"] = CategoryEducation
	c.mccCategories["8220"] = CategoryEducation
	c.mccCategories["8241"] = CategoryEducation
	c.mccCategories["8244"] = CategoryEducation
	c.mccCategories["8249"] = CategoryEducation
	c.mccCategories["8299"] = CategoryEducation
}

func (c *Categorizer) initializeMerchantPatterns() {
	patterns := map[string]string{
		// Grocery
		`(?i)(walmart|target|kroger|safeway|publix|whole\s*foods|trader\s*joe|costco|sam's\s*club|aldi|lidl|wegmans)`: CategoryGroceries,

		// Restaurants
		`(?i)(mcdonald|burger\s*king|wendy|starbucks|chipotle|subway|taco\s*bell|pizza\s*hut|domino|kfc|chick-fil-a|panera|dunkin)`: CategoryRestaurants,

		// Transportation
		`(?i)(uber|lyft|chevron|shell|exxon|bp|mobil|texaco|arco|76|marathon|citgo)`: CategoryTransportation,

		// Utilities
		`(?i)(at&t|verizon|t-mobile|sprint|comcast|xfinity|spectrum|cox|electric|water|gas\s*company|pg&e|con\s*edison)`: CategoryUtilities,

		// Entertainment
		`(?i)(netflix|spotify|hulu|disney|hbo|amazon\s*prime|apple\s*music|youtube|twitch|steam|playstation|xbox|nintendo)`: CategoryEntertainment,

		// Shopping
		`(?i)(amazon|ebay|etsy|best\s*buy|apple\s*store|home\s*depot|lowe|ikea|wayfair|nordstrom|macy|kohls)`: CategoryShopping,

		// Healthcare
		`(?i)(cvs|walgreens|rite\s*aid|pharmacy|hospital|clinic|doctor|medical|dental|optometrist|urgent\s*care)`: CategoryHealthcare,

		// Travel
		`(?i)(airbnb|booking|expedia|hotels\.com|marriott|hilton|hyatt|delta|united|american\s*airlines|southwest|jetblue)`: CategoryTravel,

		// Subscription
		`(?i)(subscription|monthly|annual\s*fee|membership|premium)`: CategorySubscription,
	}

	for pattern, category := range patterns {
		c.merchantPatterns[category] = regexp.MustCompile(pattern)
	}
}

func (c *Categorizer) initializeDescriptionRules() {
	rules := []struct {
		pattern  string
		category string
		priority int
	}{
		{`(?i)payroll|salary|direct\s*deposit|wage`, CategoryIncome, 100},
		{`(?i)interest\s*(payment|earned)`, CategoryIncome, 90},
		{`(?i)dividend`, CategoryInvestment, 90},
		{`(?i)transfer\s*(to|from)|wire\s*transfer|ach`, CategoryTransfer, 80},
		{`(?i)fee|charge|penalty|overdraft`, CategoryFees, 70},
		{`(?i)insurance|premium`, CategoryInsurance, 60},
		{`(?i)tuition|school|university|college|course`, CategoryEducation, 60},
		{`(?i)gym|fitness|health\s*club`, CategoryHealthcare, 50},
		{`(?i)parking|toll|transit`, CategoryTransportation, 50},
		{`(?i)rent|mortgage|property`, CategoryUtilities, 40},
	}

	for _, rule := range rules {
		c.descriptionRules = append(c.descriptionRules, CategorizationRule{
			Pattern:  regexp.MustCompile(rule.pattern),
			Category: rule.category,
			Priority: rule.priority,
		})
	}
}

// Categorize categorizes a transaction
func (c *Categorizer) Categorize(txn *models.Transaction) string {
	// First try MCC code if merchant is present
	if txn.Merchant != nil && txn.Merchant.MCC != "" {
		if category, ok := c.mccCategories[txn.Merchant.MCC]; ok {
			return category
		}
	}

	// Then try merchant name patterns
	if txn.Merchant != nil && txn.Merchant.Name != "" {
		for category, pattern := range c.merchantPatterns {
			if pattern.MatchString(txn.Merchant.Name) {
				return category
			}
		}
	}

	// Then try description rules
	desc := strings.ToLower(txn.Description)
	var bestMatch *CategorizationRule
	for i := range c.descriptionRules {
		rule := &c.descriptionRules[i]
		if rule.Pattern.MatchString(desc) {
			if bestMatch == nil || rule.Priority > bestMatch.Priority {
				bestMatch = rule
			}
		}
	}
	if bestMatch != nil {
		return bestMatch.Category
	}

	// Default category based on transaction type
	switch txn.Type {
	case models.TransactionTypeCredit:
		return CategoryIncome
	case models.TransactionTypeTransfer:
		return CategoryTransfer
	case models.TransactionTypeFee:
		return CategoryFees
	case models.TransactionTypeInterest:
		return CategoryIncome
	case models.TransactionTypeRefund:
		return CategoryOther
	default:
		return CategoryOther
	}
}

// AddMCCMapping adds a custom MCC to category mapping
func (c *Categorizer) AddMCCMapping(mcc, category string) {
	c.mccCategories[mcc] = category
}

// AddMerchantPattern adds a custom merchant pattern
func (c *Categorizer) AddMerchantPattern(pattern, category string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	c.merchantPatterns[category] = re
	return nil
}

// AddDescriptionRule adds a custom description rule
func (c *Categorizer) AddDescriptionRule(pattern, category string, priority int) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	c.descriptionRules = append(c.descriptionRules, CategorizationRule{
		Pattern:  re,
		Category: category,
		Priority: priority,
	})
	return nil
}

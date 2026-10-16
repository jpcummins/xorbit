package application

// Fixed text and wording rules for the operator-facing screens. Each screen
// uses a shared header, separator, title, body, and command list.
//
// Call the result the "24-word BIP-39 mnemonic code", call q "quit",
// qualify entropy-mixing claims, and say rejected dice lines were not kept.

const screenHeader = "xorbit - v0.0.0-dev"

// rule contains 80 '=' characters and follows the shared header.
const rule = "==========" + "==========" + "==========" + "==========" +
	"==========" + "==========" + "==========" + "==========" + "\n"

// Fixed App messages.
const (
	msgNotValid = "Not a valid choice. Use one of the options listed below."
	msgTooLong  = "Line too long. Nothing from it was kept."
)

// ===================================================================
// The two opening screens.
// ===================================================================

const (
	aboutTitle = "What is xorbit?"
	howTitle   = "How it works"
	riskTitle  = "Risks"
	mixTitle   = "XOR mix"
)

const (
	aboutCommands = "[n] Next: How it works\n[q] Quit"
	howCommands   = "[n] Next: Risks\n[q] Quit"
)

// aboutBody explains what the program does and why it combines sources.
const aboutBody = "xorbit creates a 24-word BIP-39 mnemonic code by mixing\n" +
	"entropy from physical dice rolls with entropy from the Linux kernel.\n"

// howBody summarizes the operator-facing flow from dice through verification.
const howBody = "1. Generate Dice entropy: roll one or more fair six-sided dice at least\n" +
	"   99 times; enter each result (1-6). Digits stream into SHA-256.\n" +
	"2. Generate RNG entropy: 256 bits come from the Linux kernel RNG.\n" +
	"3. Mix the entropy: combine two independently derived sources of\n" +
	"   entropy using bytewise XOR.\n" +
	"4. Reveal: view the 24-word BIP-39 mnemonic code.\n" +
	"5. Verify: type the 24 words from your paper, one at a time. The program\n" +
	"   checks each word against the mnemonic code before you finish.\n"

// ===================================================================
// Risks screen.
// ===================================================================

const risksBody = "This program displays sensitive private-key data that can be used to\n" +
	"unlock funds: the 24-word BIP-39 mnemonic code and the entropy shown on\n" +
	"the XOR screen. Sensitive buffers are cleared on exit.\n" +
	"Losing or mistyping your only copy may make funds unrecoverable.\n" +
	"\n" +
	"Continue only if ALL of these are true:\n" +
	"  * Offline: Wi-Fi off, cable unplugged; fresh amnesic TailsOS session.\n" +
	"  * Alone; no camera/phone/recorder; fresh terminal, no tmux/screen/script.\n" +
	"  * Trust hardware, TailsOS, Linux kernel, and exact program build;\n" +
	"    absolutely trust the methods and personally inspect this version's source.\n" +
	"  * You can roll one or more fair six-sided dice unpredictably.\n" +
	"\n" +
	"Strongly recommended: use this for only one key in a multisig wallet.\n"

const risksCommands = "[n] Next: Acknowledgement\n[q] Quit"

const mixCommands = "[n] Next: Review mnemonic code instructions\n[q] Quit"

const mixExplanation = "Mixing independent sources can help if one is unreliable.\n"

const verificationNote = "You don't need to write these down; they're presented for verification."

const (
	riskAcceptanceTitle = "Acknowledgement"
	riskAcceptanceBody  = "Source code is available to inspect, and this build is reproducible.\n" +
		"Read the code yourself; an AI assistant can help explain it. Follow the\n" +
		"binary-verification steps on the project website to verify this binary.\n" +
		"The software is provided \"as is\"; the author is not liable for loss or\n" +
		"damage.\n" +
		"\n" +
		"Don't trust, verify.\n" +
		"\n" +
		"Do you acknowledge and accept the risks?\n"
	riskAcceptanceCommands = "[a] Accept and continue\n[q] Quit"
)

// ===================================================================
// Exit commands and confirmations.
// ===================================================================

const exitCommands = "[↵] Exit"

// failedTitle and failedBody identify the fixed screen shown when an
// entropy source fails. The program then exits through its failure path.
const failedTitle = "Something went wrong"
const failedBody = "That source failed. No 24-word BIP-39 mnemonic code was created.\n"

// checkFailedBody is shown when the terminal re-scan before the
// one-time reveal fails (AGENTS.md hardening step 11): another process
// holds the terminal, so the mnemonic code is never shown.
const checkFailedBody = "The terminal security check failed. No 24-word BIP-39 mnemonic code was\n" +
	"created.\n"

// Fixed quit-confirmation text.
const (
	abortTitle       = "Quit?"
	abortBody        = "Quit now? All entropy and input collected so far will be wiped, and no\n24-word BIP-39 mnemonic code will be shown.\n"
	abortInstruction = "[y] Yes, quit\n[n] No, return to the current screen"
	revealAbortBody  = "Quit now? All entropy and input collected so far will be wiped.\n" +
		"The 24-word BIP-39 mnemonic code has already been shown and will not be\n" +
		"shown again.\n"
	revealAbortInstruction = "[y] Yes, quit\n[n] No, continue to Verify"
)

// ===================================================================
// ConfirmReveal and Reveal screens.
// ===================================================================

const (
	// The confirm screen is fixed text only: it shows nothing derived
	// from the entropy (TTY interface, item 1).
	confirmRevealTitle = "BIP-39 Mnemonic Code"
	confirmRevealBody  = "Your 24 words are ready. They are shown once, on the next screen, and\n" +
		"never again.\n" +
		"\n" +
		"Your dice rolls (hashed with SHA-256) were mixed with 256 bits from the\n" +
		"Linux kernel's random generator (a simple XOR). Words 1-23 and the first\n" +
		"3 bits of word 24 are those 256 mixed bits. The last 8 bits are a checksum.\n" +
		"The checksum is the first 8 bits of SHA-256 of the entropy. It lets\n" +
		"wallets catch a mistyped word; it adds no randomness.\n" +
		"\n" +
		"Have pen and paper ready, and make sure nothing can see your screen. A\n" +
		"verification step will follow.\n"
	confirmRevealCmds = "[n] Next: Reveal mnemonic code\n[q] Quit"

	revealTitle  = "Your 24-word BIP-39 mnemonic code"
	revealFooter = "Write the words down in order, then check each one against the screen.\n" +
		"This is the only time they are shown.\n"
	revealCommands = "[n] Next: Verify your copy\n[q] Quit"
)

// ===================================================================
// Verify and Done screens.
// ===================================================================

const (
	verifyTitle = "Confirm your 24-word BIP-39 mnemonic code"

	// verifyIntro explains what the screen is and where to look. The
	// grid shows only words typed by the operator, never words from the
	// entropy pool.
	verifyIntro = "Type each word, in order, one per line, to confirm what you wrote down.\n" +
		"Look at your paper to find each word.\n"

	// verifyCmdsDuring lists the active command while the operator is
	// still entering slots. The prompt line says `Word N of 24:` for
	// the next slot.
	verifyCmdsDuring = "[word + ↵] Check this word\n[q] Quit"

	// verifyCmdsAll is the command strip once every slot has matched.
	// The message line holds the match status; n advances to Done.
	verifyCmdsAll = "[n] Next: Finish\n[q] Quit"

	// verifyMsgPrompt is the message-line text for the next slot to
	// fill. The current 1-based slot number is computed by the App.
	verifyMsgPrompt = "Word %d of 24: enter it below."

	// verifyMsgRejected is shown after a wrong entry. The slot number
	// is the one the operator typed, so they know which row to retype.
	verifyMsgRejected = "Word %d was not correct. Type word %d again:"

	// verifyMsgAllMatch replaces the slot prompt once every slot matches.
	verifyMsgAllMatch = "All 24 slots match."

	doneTitle = "You're done"

	// doneBody reminds the operator to protect the written mnemonic and
	// explains that sensitive buffers are cleared on exit.
	doneBody = "Store the seed (your 24-word BIP-39 mnemonic code) carefully. Keep it\n" +
		"covered and never let cameras see it; anyone who sees it may be able to\n" +
		"access your funds.\n" +
		"\n" +
		"When you exit, the program clears its sensitive buffers.\n"
)

# Transform Workbench Redesign - Testing Checklist

## Pre-Flight Checks

### Build & Deploy
- [ ] Run `npm install` in `internal/web/`
- [ ] Run `npm run build` in `internal/web/`
- [ ] Verify no TypeScript compilation errors
- [ ] Verify `internal/web/static/` directory created with built files
- [ ] Build Go binary with `go build ./cmd/bibbl`
- [ ] Verify embedded web UI loads at `http://localhost:8080` (or configured port)
- [ ] Navigate to Transform Workbench tab in UI

---

## Template Gallery Testing

### Visual Display
- [ ] Gallery loads without errors
- [ ] All 7 template cards display with icons
- [ ] Cards have proper spacing and layout
- [ ] Hover effects work (card lifts up slightly)
- [ ] Icons render correctly (🔥, 🌐, ⚠️, etc.)

### Search & Filter
- [ ] Search bar accepts text input
- [ ] Typing filters templates in real-time
- [ ] Search matches title and description
- [ ] "All" category button selected by default
- [ ] Clicking category filters shows only matching templates
- [ ] Multiple category toggles work (Firewall, Security, etc.)
- [ ] "No templates found" message shows for invalid search

### Template Cards
- [ ] Each card shows: icon, title, difficulty badge, category badge
- [ ] Description text is readable and truncated properly
- [ ] "Will extract" section shows field chips
- [ ] Difficulty colors: Easy=green, Medium=orange, Advanced=red
- [ ] Clicking card navigates to builder view
- [ ] Card for "Start from Scratch" loads blank form

---

## Builder View Testing

### Navigation
- [ ] Clicking template card loads builder view
- [ ] Back button appears in top-right corner
- [ ] Clicking back shows confirmation dialog
- [ ] Confirming back returns to gallery and clears form
- [ ] Template icon and title display in header

### Form Pre-Population (Select "Palo Alto → Sentinel" template)
- [ ] Route name pre-filled: "Palo Alto → Sentinel"
- [ ] Sample log pre-filled with Palo Alto CSV example
- [ ] Filter pattern set (check if 'true' or specific pattern)
- [ ] Pipeline auto-selected (if "paloalto" pipeline exists)
- [ ] Destination auto-selected (if "sentinel" destination exists)
- [ ] Enrichment enabled checkbox checked (for GeoIP templates)

---

## Field Extraction UI Testing

### JSON Logs
- [ ] Paste JSON log in sample area
- [ ] "Accept All JSON Fields" button appears
- [ ] Clicking button sets pattern to `true`
- [ ] Help banner displays with instructions
- [ ] Pattern display shows "true" with explanation

### Text Logs (Highlight Method)
- [ ] Paste text log (e.g., Apache access log)
- [ ] Click "Choose Common Fields" button
- [ ] Modal opens with preset field list
- [ ] Click "IP Address" preset
- [ ] Pattern updates with IP regex
- [ ] Pattern display box shows current pattern
- [ ] Click "Edit" button to toggle manual pattern editing

### Visual Field Picker
- [ ] Highlight text in sample log textarea
- [ ] Selected text displays in alert box
- [ ] Enter field name (e.g., "source_ip")
- [ ] Field name sanitizes (removes special chars)
- [ ] Click "Add Field" button
- [ ] Pattern updates with named capture group
- [ ] Cancel button clears selection
- [ ] Multiple fields can be added

### Common Fields Preset Dialog
- [ ] Dialog displays 6 presets (IP, Timestamp, Email, Username, Phone, JSON)
- [ ] Each preset shows icon, name, description, example
- [ ] Clicking preset closes dialog and updates pattern
- [ ] Close button in dialog works
- [ ] ESC key closes dialog

### Pattern Display
- [ ] Current pattern shows in blue box
- [ ] Pattern truncates if too long
- [ ] "Edit" button toggles multiline textarea
- [ ] Manual edits update pattern state
- [ ] Reset button clears pattern back to 'true'

---

## Before/After Preview Testing

### Sample Input Panel (Left Side)
- [ ] Sample log textarea is editable
- [ ] Paste multiple lines (up to 20)
- [ ] Line count chip shows correct number
- [ ] Monospace font renders properly
- [ ] Placeholder text helpful and clear

### Preview Processing
- [ ] Preview auto-runs when sample or pattern changes
- [ ] Debounce delay prevents excessive API calls (400ms)
- [ ] Loading spinner shows during processing
- [ ] Preview processes first 20 lines only

### Output Panel (Right Side)
- [ ] Match count chip shows "X matched"
- [ ] Match rate alert displays:
  - Green "✓ Perfect match" if 100%
  - Blue "Matched X of Y" if partial
  - Yellow "⚠ No matches" if 0%
- [ ] Field summary chips show extracted field names
- [ ] Human-readable format displays by default (not raw JSON)
- [ ] Each matched log shows as separate card
- [ ] Field names and values displayed clearly
- [ ] Toggle button switches to raw JSON view
- [ ] Raw JSON view shows formatted JSON output

### Error Handling
- [ ] Invalid regex pattern shows error message
- [ ] API timeout shows error (not infinite spinner)
- [ ] Malformed logs don't crash preview
- [ ] Error messages are user-friendly (not stack traces)

### Edge Cases
- [ ] Empty sample input shows "Add sample logs" message
- [ ] Single line processes correctly
- [ ] 100+ lines only processes first 20 (performance)
- [ ] Special characters (unicode) handled
- [ ] Very long lines don't break layout

---

## Pipeline Builder Testing

### Pipeline Selection
- [ ] Pipeline dropdown loads available pipelines
- [ ] Selecting pipeline shows function cards
- [ ] Pipeline description displays (if available)
- [ ] "Select a pipeline..." placeholder shows when none selected
- [ ] Refresh button reloads pipeline list

### Function Cards
- [ ] Each function displays as card with icon
- [ ] Friendly name shown (not technical function name)
- [ ] Description in plain English
- [ ] Category badge displays (Parsing, Enrichment, etc.)
- [ ] Step number shows execution order
- [ ] Checkbox for enable/disable works
- [ ] Card opacity changes when disabled

### Expandable Details
- [ ] Click expand icon on function card
- [ ] Before/After example displays
- [ ] Examples use monospace font
- [ ] Collapse icon rotates smoothly
- [ ] Click again to collapse

### GeoIP Enrichment Setup
- [ ] GeoIP function shows "Setup Required" alert if no database
- [ ] Click "Setup" button opens dialog
- [ ] Dialog shows current database status
- [ ] File upload button accepts .mmdb files
- [ ] Uploading file shows progress/success
- [ ] Refresh button updates status
- [ ] Status chip shows "Database loaded ✓" when ready
- [ ] Close button dismisses dialog

### Function Mapping
- [ ] "Parse CEF" maps to friendly "Parse Security Events"
- [ ] "Parse Palo Alto" maps to friendly "Parse Palo Alto Firewall Logs"
- [ ] "geoip_enrich" maps to "Add Location Info from IP Addresses"
- [ ] "redact_pii" maps to "Remove Sensitive Data (PII)"
- [ ] Unknown functions show generic name and icon

---

## Destination Selection Testing

### Dropdown
- [ ] Destination dropdown loads available destinations
- [ ] Each option shows: Name (Type)
- [ ] "Select a destination..." placeholder shows when none selected
- [ ] Selecting destination updates state

---

## Validation & Save Testing

### Validation Messages
- [ ] Missing route name shows: "Enter a name for your route"
- [ ] Missing pipeline shows: "Select a pipeline to process logs"
- [ ] Missing destination shows: "Select where to send processed logs"
- [ ] Zero matches shows: "Your pattern doesn't match any sample logs"
- [ ] Validation message displays as alert (info or warning)

### Save Button
- [ ] Save button disabled when form invalid
- [ ] Save button enabled when all required fields filled
- [ ] Clicking save shows loading state (if applicable)
- [ ] Successful save shows green success alert at top
- [ ] Success message: "✓ Transform saved successfully! Returning to gallery..."
- [ ] Auto-redirects to gallery after 2-3 seconds

### Error Handling
- [ ] API error shows red error alert
- [ ] Error message is user-friendly
- [ ] Close button dismisses error alert
- [ ] Form remains filled after error (can retry)
- [ ] Network timeout handled gracefully

### Cancel/Back
- [ ] Cancel button returns to gallery
- [ ] Confirmation dialog shows before discarding changes
- [ ] "Yes, go back" discards form data
- [ ] "Stay" keeps form data

---

## Integration Testing

### End-to-End Flow
1. [ ] Start at gallery
2. [ ] Click "Send Firewall Logs to Microsoft Sentinel" template
3. [ ] Verify form pre-populated
4. [ ] Paste real Palo Alto log in preview
5. [ ] Verify fields extracted correctly
6. [ ] Select pipeline (or verify auto-selected)
7. [ ] Select destination (or verify auto-selected)
8. [ ] Click "Save Transform"
9. [ ] Verify success message
10. [ ] Return to gallery automatically
11. [ ] Verify new route created (check Routes tab)

### Template Coverage
- [ ] Test each of the 7 templates:
  - [ ] Palo Alto → Sentinel
  - [ ] Versa → Splunk
  - [ ] Severity Routing
  - [ ] IP Extraction + GeoIP
  - [ ] JSON Filter
  - [ ] CEF Parser
  - [ ] Start from Scratch

### Cross-Component Communication
- [ ] Changing pattern updates preview automatically
- [ ] Changing sample logs updates preview automatically
- [ ] Preview result affects validation message
- [ ] Pipeline selection loads correct functions
- [ ] Template selection populates all fields

---

## Performance Testing

### Responsiveness
- [ ] Gallery loads in < 1 second
- [ ] Template cards render without lag
- [ ] Builder view loads in < 500ms
- [ ] Preview debounce prevents API spam (400ms delay)
- [ ] Large sample logs (20+ lines) don't freeze UI

### API Calls
- [ ] Preview API called max once per 400ms (debounced)
- [ ] No duplicate API calls on mount
- [ ] API errors don't break UI state
- [ ] Refresh buttons reload data correctly

---

## Browser Compatibility

### Desktop Browsers
- [ ] Chrome/Edge (primary target)
- [ ] Firefox
- [ ] Safari (Mac)

### Responsive Design
- [ ] Template gallery grid responsive (3 cols → 2 cols → 1 col)
- [ ] Builder view readable on tablets
- [ ] Preview panels stack on narrow screens
- [ ] Buttons don't overflow on mobile

---

## Accessibility

### Keyboard Navigation
- [ ] Tab key navigates through form fields
- [ ] Enter key submits form (where appropriate)
- [ ] ESC key closes dialogs
- [ ] Focus indicators visible

### Screen Readers
- [ ] Form labels properly associated
- [ ] Buttons have aria-labels
- [ ] Error messages announced
- [ ] Success messages announced

---

## Edge Cases & Error Scenarios

### No Data Available
- [ ] No pipelines configured → Shows helpful message
- [ ] No destinations configured → Shows helpful message
- [ ] Empty pipeline (no functions) → Shows warning
- [ ] No sample logs → Shows placeholder text

### Invalid Input
- [ ] Malformed regex → Shows error, doesn't crash
- [ ] Special characters in route name → Accepts or sanitizes
- [ ] Empty string in required field → Shows validation error
- [ ] Very long route name → Truncates or scrolls

### Network Issues
- [ ] API endpoint down → Shows error message
- [ ] Slow network → Shows loading indicator
- [ ] Timeout → Cancels request and shows error

### State Management
- [ ] Browser refresh clears form (expected behavior)
- [ ] Back button navigation works
- [ ] Multiple tabs don't conflict
- [ ] Form state isolated per tab

---

## Documentation Verification

### Code Comments
- [ ] Complex logic has explanatory comments
- [ ] Component props documented with JSDoc
- [ ] Type definitions clear and accurate

### User-Facing Text
- [ ] No technical jargon in UI
- [ ] Error messages actionable ("do this to fix")
- [ ] Help text concise and clear
- [ ] Examples realistic and helpful

---

## Regression Testing (Old Functionality)

### Routes Tab
- [ ] Existing routes still load
- [ ] Can edit existing routes (might open old UI or new)
- [ ] Can delete existing routes
- [ ] Routes created in new UI appear in routes list

### Backend APIs
- [ ] `/api/v1/routes` GET still works
- [ ] `/api/v1/routes` POST still works
- [ ] `/api/v1/preview/regex` POST still works
- [ ] `/api/v1/preview/enrich` POST still works
- [ ] `/api/v1/pipelines` GET still works
- [ ] `/api/v1/destinations` GET still works

---

## Sign-Off Checklist

### Development
- [ ] All TypeScript files compile without errors
- [ ] No console errors in browser devtools
- [ ] No React warnings in console
- [ ] Code follows project style guide
- [ ] Components properly exported/imported

### Testing
- [ ] Manual testing completed (all above sections)
- [ ] At least 3 end-to-end flows tested successfully
- [ ] Edge cases handled gracefully
- [ ] Performance acceptable (no lag, fast load)

### Documentation
- [ ] TRANSFORM-WORKBENCH-REDESIGN.md reviewed
- [ ] Code comments added where needed
- [ ] User-facing changes documented

### Approval
- [ ] Product owner reviewed and approved
- [ ] UX designer reviewed (if applicable)
- [ ] Focus group testing passed (target metric: > 90% completion)
- [ ] Ready for production deployment

---

## Post-Deployment Monitoring

### Week 1
- [ ] Monitor support tickets for confusion/issues
- [ ] Track route creation rate (should increase)
- [ ] Collect user feedback
- [ ] Fix critical bugs if any

### Week 2-4
- [ ] Analyze usage metrics (which templates most popular?)
- [ ] Gather suggestions for new templates
- [ ] Plan iteration based on feedback
- [ ] Consider adding video tutorial

---

**Testing Status**: ⬜ Not Started | 🟡 In Progress | ✅ Complete | ❌ Failed

**Overall Progress**: ____%

**Sign-off**: _________________________ (Name, Date)

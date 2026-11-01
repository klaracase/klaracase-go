package agenttools

// The input schemas of the tools, written by hand in one place. A test compares the property
// names of each schema with the json tags of the params struct that the tool decodes into, so a
// new field in the SDK cannot go missing here without a failing test.

const searchDecisionsSchema = `{
  "type": "object",
  "properties": {
    "query": {"type": "string", "description": "What to look for, in German legal terms, for example \"Vermögensbetreuungspflicht Untreue\". 2 to 500 characters."},
    "court": {"type": "string", "description": "Court short form, for example \"BGH\"."},
    "branch": {"type": "string", "enum": ["ordentlich", "verwaltung", "sozial", "arbeit", "finanz", "verfassung", "patent", "unbekannt"], "description": "Court branch (the type of court, not the legal subject)."},
    "document_type": {"type": "string", "enum": ["Urteil", "Beschluss", "Verfügung", "Sonstige", "Gerichtsbescheid", "Schlussanträge"], "description": "Type of decision document."},
    "cited_norm": {"type": "string", "description": "Only decisions that cite this norm, for example \"§ 906 BGB\"."},
    "cited_statutes": {"type": "array", "items": {"type": "string", "enum": ["StGB", "StPO", "InsO", "DSGVO", "GDPR", "BDSG", "BauGB", "BBauG", "BauNVO", "StVG", "StVO"]}, "description": "Only decisions that cite any section of these statutes."},
    "date_from": {"type": "string", "description": "Earliest decision date, as YYYY-MM-DD."},
    "date_to": {"type": "string", "description": "Latest decision date, as YYYY-MM-DD. Not before date_from."},
    "mode": {"type": "string", "enum": ["keyword", "semantic", "hybrid"], "description": "Search mode. Leave it out for the default, hybrid."},
    "limit": {"type": "integer", "description": "How many results, 1 to 20."},
    "page": {"type": "integer", "description": "Result page, 1 to 10."},
    "bridge": {"type": "boolean", "description": "Let the server add German legal register terms to the query. The default is true."}
  },
  "required": ["query"],
  "additionalProperties": false
}`

const crossTypeSearchSchema = `{
  "type": "object",
  "properties": {
    "query": {"type": "string", "description": "What to look for, in German legal terms. 2 to 500 characters."},
    "jurisdiction": {"type": "string", "enum": ["DE", "EU", "XI"], "description": "Narrows the statute results only: DE is German law, EU is EU law, XI is international law."},
    "court": {"type": "string", "description": "Court short form for the decision results, for example \"BGH\"."},
    "branch": {"type": "string", "enum": ["ordentlich", "verwaltung", "sozial", "arbeit", "finanz", "verfassung", "patent", "unbekannt"], "description": "Court branch for the decision results."},
    "date_from": {"type": "string", "description": "Earliest decision date, as YYYY-MM-DD."},
    "date_to": {"type": "string", "description": "Latest decision date, as YYYY-MM-DD. Not before date_from."},
    "mode": {"type": "string", "enum": ["keyword", "semantic", "hybrid"], "description": "Search mode. Leave it out for the default, hybrid."},
    "document_type": {"type": "string", "enum": ["Urteil", "Beschluss", "Verfügung", "Sonstige", "Gerichtsbescheid", "Schlussanträge"], "description": "Type of decision document."},
    "cited_norm": {"type": "string", "description": "Only decisions that cite this norm, for example \"§ 906 BGB\"."},
    "cited_statutes": {"type": "array", "items": {"type": "string", "enum": ["StGB", "StPO", "InsO", "DSGVO", "GDPR", "BDSG", "BauGB", "BBauG", "BauNVO", "StVG", "StVO"]}, "description": "Only decisions that cite any section of these statutes."},
    "limit": {"type": "integer", "description": "How many results per section, 1 to 10."},
    "bridge": {"type": "boolean", "description": "Let the server add German legal register terms to the query. The default is true."}
  },
  "required": ["query"],
  "additionalProperties": false
}`

const getDecisionSchema = `{
  "type": "object",
  "properties": {
    "id": {"type": "string", "description": "The decision: its id from a search result, its Aktenzeichen (for example \"VIII ZR 271/17\") or its ECLI."},
    "max_chars": {"type": "integer", "description": "Most characters of text to return. The default here is 4000. The least the server accepts is 500."},
    "rn_from": {"type": "integer", "description": "First Randnummer (margin number) to return."},
    "rn_to": {"type": "integer", "description": "Last Randnummer to return. Not before rn_from."},
    "section": {"type": "string", "enum": ["Tenor", "Leitsatz", "Tatbestand", "Gründe"], "description": "Return only this section of the decision."},
    "court": {"type": "string", "description": "Court short form. Use it when the Aktenzeichen belongs to more than one decision."},
    "date": {"type": "string", "description": "Decision date as YYYY-MM-DD. Use it when the Aktenzeichen belongs to more than one decision."}
  },
  "required": ["id"],
  "additionalProperties": false
}`

const getNormSchema = `{
  "type": "object",
  "properties": {
    "reference": {"type": "string", "description": "The norm, for example \"§ 535 BGB\" or \"Art. 6 DSGVO\". One norm, not a range."},
    "jurisdiction": {"type": "string", "enum": ["DE", "EU"], "description": "Only needed when the abbreviation belongs to both German and EU law."}
  },
  "required": ["reference"],
  "additionalProperties": false
}`

const checkCitationsSchema = `{
  "type": "object",
  "properties": {
    "text": {"type": "string", "description": "The German legal text whose citations to check. 50 to 20000 characters. Send either text or document_id."},
    "document_id": {"type": "string", "description": "The id of an uploaded document that is ready. Send either text or document_id."},
    "reference_date": {"type": "string", "description": "The date of the text as YYYY-MM-DD. It dates statute citations that name no date."}
  },
  "additionalProperties": false
}`

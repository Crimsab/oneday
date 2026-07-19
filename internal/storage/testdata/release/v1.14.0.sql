/* WARNING: Script requires that SQLITE_DBCONFIG_DEFENSIVE be disabled */
PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;
CREATE TABLE schema_version (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
INSERT INTO schema_version VALUES(1,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(2,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(3,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(4,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(5,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(6,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(7,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(8,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(9,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(10,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(11,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(12,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(13,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(14,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(15,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(16,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(17,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(18,'2026-07-19 01:55:13');
INSERT INTO schema_version VALUES(19,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(20,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(21,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(22,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(23,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(24,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(25,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(26,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(27,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(28,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(29,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(30,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(31,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(32,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(33,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(34,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(35,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(36,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(37,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(38,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(39,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(40,'2026-07-19 01:55:14');
INSERT INTO schema_version VALUES(41,'2026-07-19 01:55:14');
CREATE TABLE stories (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	setting_json TEXT NOT NULL DEFAULT '{}',
	stats_schema_json TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
, description TEXT NOT NULL DEFAULT '', genre TEXT NOT NULL DEFAULT '', tone TEXT NOT NULL DEFAULT '', language TEXT NOT NULL DEFAULT '', writing_style TEXT NOT NULL DEFAULT '', prompt_directives TEXT NOT NULL DEFAULT '', is_archived INTEGER NOT NULL DEFAULT 0, revision INTEGER NOT NULL DEFAULT 0, active_branch_id TEXT NOT NULL DEFAULT '');
INSERT INTO stories VALUES('release-story','Release fixture','{"genre":"mystery"}','{}','2026-07-19 01:55:14','2026-07-19 01:55:14','','','','','','',0,7,'f61f60b8-0f99-44ad-9d3e-27652dc909ab');
CREATE TABLE characters (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	background TEXT NOT NULL DEFAULT '',
	stats_json TEXT NOT NULL DEFAULT '{}',
	traits_json TEXT NOT NULL DEFAULT '[]',
	skills_json TEXT NOT NULL DEFAULT '[]',
	inventory_json TEXT NOT NULL DEFAULT '[]',
	known_recipes_json TEXT NOT NULL DEFAULT '[]',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO characters VALUES('release-hero','release-story','Mara','','{}','[]','[]','[]','[]','2026-07-19 01:55:14','2026-07-19 01:55:14');
CREATE TABLE npcs (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	role TEXT NOT NULL DEFAULT '',
	personality_json TEXT NOT NULL DEFAULT '{}',
	private_thoughts TEXT NOT NULL DEFAULT '',
	desires TEXT NOT NULL DEFAULT '',
	disposition INTEGER NOT NULL DEFAULT 0,
	is_alive INTEGER NOT NULL DEFAULT 1,
	first_appeared_turn INTEGER NOT NULL DEFAULT 0,
	can_help INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
, appearance TEXT NOT NULL DEFAULT '', notes_on_protagonist TEXT NOT NULL DEFAULT '[]', last_seen_turn INTEGER NOT NULL DEFAULT 0, relationship_json TEXT NOT NULL DEFAULT '{}', nemesis_json TEXT NOT NULL DEFAULT '{}', discovery_json TEXT NOT NULL DEFAULT '{}', canonical_entity_id TEXT NOT NULL DEFAULT '');
INSERT INTO npcs VALUES('release-npc','release-story','Keeper','','{}','','',0,1,0,0,'2026-07-19 01:55:14','2026-07-19 01:55:14','','[]',0,'{}','{}','{}','release-npc');
CREATE TABLE world_state (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL UNIQUE REFERENCES stories(id) ON DELETE CASCADE,
	current_location TEXT NOT NULL DEFAULT '',
	known_locations_json TEXT NOT NULL DEFAULT '[]',
	global_events_json TEXT NOT NULL DEFAULT '[]',
	faction_standings_json TEXT NOT NULL DEFAULT '{}',
	current_chapter INTEGER NOT NULL DEFAULT 1,
	current_turn INTEGER NOT NULL DEFAULT 0,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
, story_hooks_json TEXT NOT NULL DEFAULT '[]', world_reactions_json TEXT NOT NULL DEFAULT '[]', player_guidance_json TEXT NOT NULL DEFAULT '[]', fronts_json TEXT NOT NULL DEFAULT '[]', investigation_board_json TEXT NOT NULL DEFAULT '{}', project_clocks_json TEXT NOT NULL DEFAULT '{}', character_timeline_json TEXT NOT NULL DEFAULT '{}', scene_contract_json TEXT NOT NULL DEFAULT '{}', current_location_id TEXT NOT NULL DEFAULT '');
INSERT INTO world_state VALUES(NULL,'release-story','Harbor','[]','[]','{}',1,7,'2026-07-19 01:55:14','[]','[]','[]','[]','{}','{}','{}','{}','');
CREATE TABLE sessions (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	ended_at DATETIME,
	summary TEXT NOT NULL DEFAULT ''
, branch_id TEXT NOT NULL DEFAULT '', source_commit_id TEXT NOT NULL DEFAULT '');
INSERT INTO sessions VALUES('release-session','release-story','2026-07-19 01:55:14',NULL,'fixture session','f61f60b8-0f99-44ad-9d3e-27652dc909ab','4ac7f24a-1a0c-4f0c-b95b-db808871bd05');
CREATE TABLE chapters (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	chapter_number INTEGER NOT NULL,
	title TEXT NOT NULL DEFAULT '',
	summary TEXT NOT NULL DEFAULT '',
	start_turn INTEGER NOT NULL,
	end_turn INTEGER,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP, branch_id TEXT NOT NULL DEFAULT '', source_commit_id TEXT NOT NULL DEFAULT '',
	UNIQUE(story_id, chapter_number)
);
CREATE TABLE achievements (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	category TEXT NOT NULL DEFAULT 'story',
	rarity TEXT NOT NULL DEFAULT 'common',
	context TEXT NOT NULL DEFAULT '',
	earned_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE saves (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	name TEXT NOT NULL DEFAULT 'autosave',
	turn INTEGER NOT NULL,
	chapter INTEGER NOT NULL,
	location TEXT NOT NULL DEFAULT '',
	character_json TEXT NOT NULL,
	world_state_json TEXT NOT NULL,
	session_id TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
, metadata_json TEXT NOT NULL DEFAULT '{}', branch_id TEXT NOT NULL DEFAULT '', source_commit_id TEXT NOT NULL DEFAULT '');
INSERT INTO saves VALUES('release-save','release-story','Before the door',7,1,'Harbor','{}','{}','release-session','2026-07-19 01:55:14','{}','f61f60b8-0f99-44ad-9d3e-27652dc909ab','4ac7f24a-1a0c-4f0c-b95b-db808871bd05');
CREATE TABLE rag_chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    chunk_type TEXT NOT NULL DEFAULT 'summary',
    turn_start INTEGER NOT NULL DEFAULT 0,
    turn_end INTEGER NOT NULL DEFAULT 0,
    embedding BLOB,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
, branch_id TEXT NOT NULL DEFAULT '', source_commit_id TEXT NOT NULL DEFAULT '', embedding_norm REAL NOT NULL DEFAULT 0);
CREATE TABLE combat_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    session_id TEXT NOT NULL,
    enemy_name TEXT NOT NULL,
    enemy_hp INTEGER NOT NULL,
    turns INTEGER NOT NULL,
    victory BOOLEAN NOT NULL,
    defeat_outcome TEXT NOT NULL DEFAULT '',
    player_hp_start INTEGER NOT NULL,
    player_hp_end INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
, branch_id TEXT NOT NULL DEFAULT '', source_commit_id TEXT NOT NULL DEFAULT '');
CREATE TABLE chat_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	turn INTEGER NOT NULL DEFAULT 0,
	role TEXT NOT NULL CHECK(role IN ('user', 'assistant', 'system', 'narrator')),
	content TEXT NOT NULL,
	message_type TEXT NOT NULL DEFAULT 'narrative' CHECK(message_type IN ('narrative', 'combat', 'crafting', 'dialogue', 'narrator', 'combat_summary')),
	metadata_json TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
, branch_id TEXT NOT NULL DEFAULT '', source_commit_id TEXT NOT NULL DEFAULT '');
INSERT INTO chat_messages VALUES(1,'release-session','release-story',7,'assistant','The lighthouse answers.','narrative','{}','2026-07-19 01:55:14','f61f60b8-0f99-44ad-9d3e-27652dc909ab','4ac7f24a-1a0c-4f0c-b95b-db808871bd05');
CREATE TABLE turn_idempotency (
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	idempotency_key TEXT NOT NULL,
	events_json TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP, status TEXT NOT NULL DEFAULT 'committed', owner TEXT NOT NULL DEFAULT '', locked_until TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL DEFAULT '', error TEXT NOT NULL DEFAULT '', request_hash TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (story_id, idempotency_key)
);
CREATE TABLE story_turn_locks (
	story_id TEXT PRIMARY KEY REFERENCES stories(id) ON DELETE CASCADE,
	owner TEXT NOT NULL,
	acquired_at TEXT NOT NULL,
	locked_until TEXT NOT NULL
);
CREATE TABLE story_visual_profiles (
	story_id TEXT PRIMARY KEY REFERENCES stories(id) ON DELETE CASCADE,
	world_style_prompt TEXT NOT NULL DEFAULT '',
	character_style_prompt TEXT NOT NULL DEFAULT '',
	negative_prompt TEXT NOT NULL DEFAULT '',
	palette TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE visual_asset_versions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	kind TEXT NOT NULL,
	subject TEXT NOT NULL,
	url TEXT NOT NULL DEFAULT '',
	file_path TEXT NOT NULL DEFAULT '',
	prompt TEXT NOT NULL DEFAULT '',
	negative_prompt TEXT NOT NULL DEFAULT '',
	provider TEXT NOT NULL DEFAULT '',
	turn INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
, branch_id TEXT NOT NULL DEFAULT '', source_commit_id TEXT NOT NULL DEFAULT '', canonical_entity_id TEXT NOT NULL DEFAULT '', canonical_location_id TEXT NOT NULL DEFAULT '', form_id TEXT NOT NULL DEFAULT '', appearance_fingerprint TEXT NOT NULL DEFAULT '', profile_revision_id TEXT, canon_status TEXT NOT NULL DEFAULT 'draft', map_scope_kind TEXT NOT NULL DEFAULT '', map_scope_id TEXT NOT NULL DEFAULT '', parent_version_id INTEGER REFERENCES visual_asset_versions(id), operation_id TEXT, mask_id TEXT);
CREATE TABLE visual_generation_jobs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
	attempts INTEGER NOT NULL DEFAULT 0,
	max_attempts INTEGER NOT NULL DEFAULT 3,
	locked_until TEXT NOT NULL DEFAULT '',
	request_payload_json TEXT NOT NULL DEFAULT '{}',
	error TEXT NOT NULL DEFAULT '',
	provider TEXT NOT NULL DEFAULT '',
	started_at TEXT NOT NULL DEFAULT '',
	finished_at TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
, branch_id TEXT NOT NULL DEFAULT '', source_commit_id TEXT NOT NULL DEFAULT '', canonical_entity_id TEXT NOT NULL DEFAULT '', canonical_location_id TEXT NOT NULL DEFAULT '', form_id TEXT NOT NULL DEFAULT '', appearance_fingerprint TEXT NOT NULL DEFAULT '', profile_revision_id TEXT, map_scope_kind TEXT NOT NULL DEFAULT '', map_scope_id TEXT NOT NULL DEFAULT '');
CREATE TABLE story_branches (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	fork_commit_id TEXT,
	head_commit_id TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id, name)
);
INSERT INTO story_branches VALUES('f61f60b8-0f99-44ad-9d3e-27652dc909ab','release-story','main',NULL,'4ac7f24a-1a0c-4f0c-b95b-db808871bd05','2026-07-19 01:55:14.769215977 +0000 UTC','2026-07-19 01:55:14.769215977 +0000 UTC');
CREATE TABLE turn_commits (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	parent_commit_id TEXT REFERENCES turn_commits(id),
	canonical_turn INTEGER NOT NULL,
	story_revision INTEGER NOT NULL,
	payload_hash TEXT NOT NULL,
	kind TEXT NOT NULL DEFAULT 'turn',
	message TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO turn_commits VALUES('4ac7f24a-1a0c-4f0c-b95b-db808871bd05','release-story','f61f60b8-0f99-44ad-9d3e-27652dc909ab',NULL,0,7,'','root','','2026-07-19 01:55:14.769215977 +0000 UTC');
CREATE TABLE turn_snapshots (
	commit_id TEXT PRIMARY KEY REFERENCES turn_commits(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	format_version INTEGER NOT NULL,
	payload_json TEXT NOT NULL,
	payload_hash TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE canonical_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	commit_id TEXT NOT NULL REFERENCES turn_commits(id) ON DELETE CASCADE,
	sequence INTEGER NOT NULL,
	event_type TEXT NOT NULL,
	payload_json TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(commit_id, sequence)
);
CREATE TABLE save_bookmarks (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	save_id TEXT REFERENCES saves(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(save_id)
);
CREATE TABLE generation_traces (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	kind TEXT NOT NULL,
	request_id TEXT NOT NULL DEFAULT '',
	payload_json TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE audio_artifacts (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	kind TEXT NOT NULL DEFAULT 'narration',
	entity_id TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending',
	url TEXT NOT NULL DEFAULT '',
	file_path TEXT NOT NULL DEFAULT '',
	metadata_json TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE canonical_entities (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	entity_kind TEXT NOT NULL,
	canonical_name TEXT NOT NULL DEFAULT '',
	lifecycle_status TEXT NOT NULL DEFAULT 'active',
	profile_json TEXT NOT NULL DEFAULT '{}',
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE entity_aliases (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	entity_id TEXT NOT NULL REFERENCES canonical_entities(id) ON DELETE CASCADE,
	alias TEXT NOT NULL,
	alias_kind TEXT NOT NULL DEFAULT 'known',
	visibility TEXT NOT NULL DEFAULT 'private',
	valid_from_turn INTEGER NOT NULL DEFAULT 0,
	valid_to_turn INTEGER,
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(entity_id,alias,alias_kind,branch_id)
);
CREATE TABLE identity_claims (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	subject_entity_id TEXT NOT NULL REFERENCES canonical_entities(id) ON DELETE CASCADE,
	claimed_entity_id TEXT REFERENCES canonical_entities(id),
	observer_entity_id TEXT REFERENCES canonical_entities(id),
	label TEXT NOT NULL,
	claim_kind TEXT NOT NULL,
	status TEXT NOT NULL CHECK(status IN ('rumor','observed','confirmed','contradicted','retracted','reverified')),
	confidence REAL NOT NULL CHECK(confidence>=0 AND confidence<=1),
	visibility TEXT NOT NULL DEFAULT 'private',
	evidence_json TEXT NOT NULL DEFAULT '[]',
	learned_turn INTEGER NOT NULL DEFAULT 0,
	valid_from_world_time TEXT NOT NULL DEFAULT '',
	valid_to_world_time TEXT NOT NULL DEFAULT '',
	supersedes_claim_id TEXT REFERENCES identity_claims(id),
	contradicts_claim_id TEXT REFERENCES identity_claims(id),
	retracts_claim_id TEXT REFERENCES identity_claims(id),
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE entity_forms (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	entity_id TEXT NOT NULL REFERENCES canonical_entities(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	form_kind TEXT NOT NULL,
	body_entity_id TEXT REFERENCES canonical_entities(id),
	appearance_json TEXT NOT NULL DEFAULT '{}',
	valid_from_turn INTEGER NOT NULL DEFAULT 0,
	valid_to_turn INTEGER,
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE entity_controller_events (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	form_id TEXT NOT NULL REFERENCES entity_forms(id) ON DELETE CASCADE,
	controller_entity_id TEXT NOT NULL REFERENCES canonical_entities(id),
	control_kind TEXT NOT NULL CHECK(control_kind IN ('self','possession','body_theft','puppetry','shared','unknown')),
	status TEXT NOT NULL CHECK(status IN ('started','ended','disputed')),
	turn INTEGER NOT NULL,
	world_time TEXT NOT NULL DEFAULT '',
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE entity_lifecycle_events (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	entity_id TEXT NOT NULL REFERENCES canonical_entities(id) ON DELETE CASCADE,
	status TEXT NOT NULL,
	turn INTEGER NOT NULL,
	world_time TEXT NOT NULL DEFAULT '',
	reason TEXT NOT NULL DEFAULT '',
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE character_facts (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	subject_entity_id TEXT NOT NULL REFERENCES canonical_entities(id) ON DELETE CASCADE,
	predicate TEXT NOT NULL,
	object_json TEXT NOT NULL,
	source_entity_id TEXT REFERENCES canonical_entities(id),
	source_event_id TEXT NOT NULL DEFAULT '',
	observer_entity_id TEXT REFERENCES canonical_entities(id),
	learned_turn INTEGER NOT NULL,
	valid_from_world_time TEXT NOT NULL DEFAULT '',
	valid_to_world_time TEXT NOT NULL DEFAULT '',
	confidence REAL NOT NULL CHECK(confidence>=0 AND confidence<=1),
	visibility TEXT NOT NULL DEFAULT 'private',
	supersedes_fact_id TEXT REFERENCES character_facts(id),
	contradicts_fact_id TEXT REFERENCES character_facts(id),
	retracts_fact_id TEXT REFERENCES character_facts(id),
	evidence_json TEXT NOT NULL DEFAULT '[]',
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE entity_field_locks (
	entity_id TEXT NOT NULL REFERENCES canonical_entities(id) ON DELETE CASCADE,
	field_path TEXT NOT NULL,
	lock_kind TEXT NOT NULL CHECK(lock_kind IN ('profile','visual')),
	locked_value_json TEXT NOT NULL,
	locked_by TEXT NOT NULL DEFAULT 'player',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(entity_id,field_path,lock_kind)
);
CREATE TABLE factions (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	profile_json TEXT NOT NULL DEFAULT '{}',
	visibility TEXT NOT NULL DEFAULT 'private',
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id,name,branch_id)
);
CREATE TABLE faction_membership_events (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	faction_id TEXT NOT NULL REFERENCES factions(id) ON DELETE CASCADE,
	entity_id TEXT NOT NULL REFERENCES canonical_entities(id) ON DELETE CASCADE,
	status TEXT NOT NULL CHECK(status IN ('joined','left','expelled','rumored','confirmed')),
	role TEXT NOT NULL DEFAULT '',
	visibility TEXT NOT NULL DEFAULT 'private',
	turn INTEGER NOT NULL,
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE faction_relationship_events (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	source_faction_id TEXT NOT NULL REFERENCES factions(id),
	target_faction_id TEXT NOT NULL REFERENCES factions(id),
	delta INTEGER NOT NULL CHECK(delta>=-100 AND delta<=100),
	reason TEXT NOT NULL,
	turn INTEGER NOT NULL,
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE reputation_events (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	faction_id TEXT NOT NULL REFERENCES factions(id),
	entity_id TEXT NOT NULL REFERENCES canonical_entities(id),
	delta INTEGER NOT NULL CHECK(delta>=-100 AND delta<=100),
	reason TEXT NOT NULL,
	source_event_id TEXT NOT NULL DEFAULT '',
	visibility TEXT NOT NULL DEFAULT 'player',
	turn INTEGER NOT NULL,
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE regions (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,name TEXT NOT NULL,parent_region_id TEXT REFERENCES regions(id),visibility TEXT NOT NULL DEFAULT 'private',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id), region_kind TEXT NOT NULL DEFAULT 'region',UNIQUE(story_id,name,branch_id));
CREATE TABLE locations (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,canonical_name TEXT NOT NULL,region_id TEXT REFERENCES regions(id),parent_location_id TEXT REFERENCES locations(id),description TEXT NOT NULL DEFAULT '',discovery_state TEXT NOT NULL DEFAULT 'unknown',discovered_turn INTEGER NOT NULL DEFAULT 0,visibility TEXT NOT NULL DEFAULT 'player',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),created_at DATETIME DEFAULT CURRENT_TIMESTAMP, location_kind TEXT NOT NULL DEFAULT 'place',UNIQUE(story_id,canonical_name,branch_id));
CREATE TABLE location_aliases (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,location_id TEXT NOT NULL REFERENCES locations(id) ON DELETE CASCADE,alias TEXT NOT NULL,visibility TEXT NOT NULL DEFAULT 'player',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),UNIQUE(location_id,alias,branch_id));
CREATE TABLE location_edges (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,from_location_id TEXT NOT NULL REFERENCES locations(id),to_location_id TEXT NOT NULL REFERENCES locations(id),direction TEXT NOT NULL DEFAULT '',travel_minutes INTEGER NOT NULL DEFAULT 0,conditions_json TEXT NOT NULL DEFAULT '{}',valid_from_world_time TEXT NOT NULL DEFAULT '',valid_to_world_time TEXT NOT NULL DEFAULT '',visibility TEXT NOT NULL DEFAULT 'private',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id), travel_mode TEXT NOT NULL DEFAULT 'travel', bidirectional INTEGER NOT NULL DEFAULT 0 CHECK(bidirectional IN (0,1)));
CREATE TABLE entity_position_events (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,entity_id TEXT NOT NULL,location_id TEXT NOT NULL REFERENCES locations(id),event_kind TEXT NOT NULL,turn INTEGER NOT NULL,world_time TEXT NOT NULL DEFAULT '',visibility TEXT NOT NULL DEFAULT 'player',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE world_calendars (story_id TEXT PRIMARY KEY REFERENCES stories(id) ON DELETE CASCADE,name TEXT NOT NULL,config_json TEXT NOT NULL);
CREATE TABLE world_clocks (story_id TEXT PRIMARY KEY REFERENCES stories(id) ON DELETE CASCADE,calendar_story_id TEXT NOT NULL REFERENCES world_calendars(story_id),day INTEGER NOT NULL DEFAULT 0,minute_of_day INTEGER NOT NULL DEFAULT 0,display_text TEXT NOT NULL DEFAULT '',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE world_time_events (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,delta_minutes INTEGER NOT NULL,reason TEXT NOT NULL,turn INTEGER NOT NULL,from_day INTEGER NOT NULL,from_minute INTEGER NOT NULL,to_day INTEGER NOT NULL,to_minute INTEGER NOT NULL,branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE weather_states (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,region_id TEXT REFERENCES regions(id),location_id TEXT REFERENCES locations(id),weather_kind TEXT NOT NULL,intensity TEXT NOT NULL DEFAULT '',description TEXT NOT NULL DEFAULT '',valid_from_day INTEGER NOT NULL,valid_from_minute INTEGER NOT NULL,valid_to_day INTEGER,valid_to_minute INTEGER,visibility TEXT NOT NULL DEFAULT 'player',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE canonical_world_events (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,event_kind TEXT NOT NULL,title TEXT NOT NULL,details_json TEXT NOT NULL DEFAULT '{}',location_id TEXT REFERENCES locations(id),faction_id TEXT REFERENCES factions(id),entity_id TEXT REFERENCES canonical_entities(id),caused_by_event_id TEXT REFERENCES canonical_world_events(id),turn INTEGER NOT NULL,world_day INTEGER NOT NULL,world_minute INTEGER NOT NULL,visibility TEXT NOT NULL DEFAULT 'private',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE world_thread_events (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,thread_id TEXT NOT NULL,title TEXT NOT NULL,status TEXT NOT NULL,pressure INTEGER NOT NULL DEFAULT 0,details_json TEXT NOT NULL DEFAULT '{}',visibility TEXT NOT NULL DEFAULT 'private',turn INTEGER NOT NULL,branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE challenge_runs (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	session_id TEXT NOT NULL DEFAULT '',
	turn INTEGER NOT NULL,
	protocol_version INTEGER NOT NULL,
	definition_json TEXT NOT NULL CHECK(json_valid(definition_json)),
	instance_json TEXT NOT NULL CHECK(json_valid(instance_json)),
	input_json TEXT NOT NULL CHECK(json_valid(input_json)),
	resolution_json TEXT NOT NULL CHECK(json_valid(resolution_json)),
	outcome_json TEXT NOT NULL CHECK(json_valid(outcome_json)),
	degree TEXT NOT NULL CHECK(degree IN ('critical_success','full_success','success_with_cost','failure_with_progress','hard_failure','catastrophe')),
	difficulty INTEGER NOT NULL,
	seed INTEGER NOT NULL,
	roll INTEGER NOT NULL,
	total INTEGER NOT NULL,
	margin INTEGER NOT NULL,
	modifiers_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(modifiers_json)),
	timing_policy_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(timing_policy_json)),
	costs_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(costs_json)),
	state_deltas_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(state_deltas_json)),
	branch_id TEXT NOT NULL,
	source_commit_id TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE prompt_profiles (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	description TEXT NOT NULL DEFAULT '',
	redaction_policy TEXT NOT NULL DEFAULT 'secrets_and_reasoning',
	retention_days INTEGER NOT NULL DEFAULT 30 CHECK(retention_days BETWEEN 1 AND 3650),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE prompt_profile_revisions (
	id TEXT PRIMARY KEY,
	profile_id TEXT NOT NULL REFERENCES prompt_profiles(id) ON DELETE CASCADE,
	version INTEGER NOT NULL CHECK(version > 0),
	template_version TEXT NOT NULL DEFAULT '',
	prompt_hash TEXT NOT NULL,
	response_schema_hash TEXT NOT NULL DEFAULT '',
	config_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(config_json)),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(profile_id, version),
	UNIQUE(profile_id, prompt_hash, response_schema_hash)
);
CREATE TABLE generation_runs (
	id TEXT PRIMARY KEY,
	trace_id TEXT NOT NULL,
	parent_run_id TEXT REFERENCES generation_runs(id),
	story_id TEXT NOT NULL DEFAULT '',
	branch_id TEXT NOT NULL DEFAULT '',
	source_commit_id TEXT NOT NULL DEFAULT '',
	message_id INTEGER,
	stage TEXT NOT NULL,
	status TEXT NOT NULL CHECK(status IN ('running','succeeded','failed','cancelled')),
	prompt_revision_id TEXT REFERENCES prompt_profile_revisions(id),
	prompt_hash TEXT NOT NULL,
	request_config_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(request_config_json)),
	requested_streaming INTEGER NOT NULL DEFAULT 0 CHECK(requested_streaming IN (0,1)),
	observed_streaming INTEGER NOT NULL DEFAULT 0 CHECK(observed_streaming IN (0,1)),
	input_tokens INTEGER NOT NULL DEFAULT 0,
	output_tokens INTEGER NOT NULL DEFAULT 0,
	reasoning_tokens INTEGER NOT NULL DEFAULT 0,
	cached_input_tokens INTEGER NOT NULL DEFAULT 0,
	total_tokens INTEGER NOT NULL DEFAULT 0,
	cost_usd REAL NOT NULL DEFAULT 0,
	ttft_ms INTEGER NOT NULL DEFAULT 0,
	duration_ms INTEGER NOT NULL DEFAULT 0,
	error_class TEXT NOT NULL DEFAULT '',
	metadata_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(metadata_json)),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	finished_at DATETIME
);
CREATE TABLE generation_attempts (
	id TEXT PRIMARY KEY,
	run_id TEXT NOT NULL REFERENCES generation_runs(id) ON DELETE CASCADE,
	sequence INTEGER NOT NULL CHECK(sequence > 0),
	provider TEXT NOT NULL,
	requested_model TEXT NOT NULL DEFAULT '',
	resolved_model TEXT NOT NULL DEFAULT '',
	reasoning_config_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(reasoning_config_json)),
	requested_streaming INTEGER NOT NULL DEFAULT 0 CHECK(requested_streaming IN (0,1)),
	observed_streaming INTEGER NOT NULL DEFAULT 0 CHECK(observed_streaming IN (0,1)),
	status TEXT NOT NULL CHECK(status IN ('running','succeeded','failed','cancelled')),
	ttft_ms INTEGER NOT NULL DEFAULT 0,
	duration_ms INTEGER NOT NULL DEFAULT 0,
	input_tokens INTEGER NOT NULL DEFAULT 0,
	output_tokens INTEGER NOT NULL DEFAULT 0,
	reasoning_tokens INTEGER NOT NULL DEFAULT 0,
	cached_input_tokens INTEGER NOT NULL DEFAULT 0,
	total_tokens INTEGER NOT NULL DEFAULT 0,
	cost_usd REAL NOT NULL DEFAULT 0,
	retry_reason TEXT NOT NULL DEFAULT '',
	error_class TEXT NOT NULL DEFAULT '',
	error_summary TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	finished_at DATETIME,
	UNIQUE(run_id, sequence)
);
CREATE TABLE generation_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	run_id TEXT NOT NULL REFERENCES generation_runs(id) ON DELETE CASCADE,
	attempt_id TEXT REFERENCES generation_attempts(id) ON DELETE CASCADE,
	event_type TEXT NOT NULL,
	payload_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(payload_json)),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE visual_profile_revisions (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	revision INTEGER NOT NULL CHECK(revision > 0),
	world_style_prompt TEXT NOT NULL DEFAULT '',
	character_style_prompt TEXT NOT NULL DEFAULT '',
	negative_prompt TEXT NOT NULL DEFAULT '',
	palette TEXT NOT NULL DEFAULT '',
	fingerprint TEXT NOT NULL,
	branch_id TEXT NOT NULL,
	source_commit_id TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id, branch_id, revision),
	UNIQUE(story_id, branch_id, fingerprint)
);
CREATE TABLE IF NOT EXISTS "visual_assets" (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	kind TEXT NOT NULL,
	subject TEXT NOT NULL,
	entity_id TEXT NOT NULL DEFAULT '',
	canonical_entity_id TEXT NOT NULL DEFAULT '',
	canonical_location_id TEXT NOT NULL DEFAULT '',
	form_id TEXT NOT NULL DEFAULT '',
	lineage_key TEXT NOT NULL,
	appearance_fingerprint TEXT NOT NULL,
	profile_revision_id TEXT REFERENCES visual_profile_revisions(id),
	canon_status TEXT NOT NULL DEFAULT 'draft',
	gate_state TEXT NOT NULL DEFAULT 'eligible',
	gate_reason TEXT NOT NULL DEFAULT '',
	generation_eligible INTEGER NOT NULL DEFAULT 1 CHECK(generation_eligible IN (0,1)),
	prompt TEXT NOT NULL DEFAULT '',
	negative_prompt TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','queued','running','ready','failed')),
	url TEXT NOT NULL DEFAULT '',
	file_path TEXT NOT NULL DEFAULT '',
	provider TEXT NOT NULL DEFAULT '',
	source TEXT NOT NULL DEFAULT '',
	error TEXT NOT NULL DEFAULT '',
	turn INTEGER NOT NULL DEFAULT 0,
	branch_id TEXT NOT NULL,
	source_commit_id TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP, map_scope_kind TEXT NOT NULL DEFAULT '', map_scope_id TEXT NOT NULL DEFAULT '',
	UNIQUE(story_id, branch_id, lineage_key)
);
CREATE TABLE visual_asset_branch_overrides (
	asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL,
	source_commit_id TEXT NOT NULL,
	prompt_override TEXT NOT NULL DEFAULT '',
	negative_prompt_override TEXT NOT NULL DEFAULT '',
	gate_state TEXT NOT NULL DEFAULT '',
	gate_reason TEXT NOT NULL DEFAULT '',
	generation_eligible INTEGER CHECK(generation_eligible IN (0,1)),
	status_override TEXT NOT NULL DEFAULT '',
	error_override TEXT NOT NULL DEFAULT '',
	provider_override TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(asset_id,branch_id)
);
CREATE TABLE visual_asset_selection_states (
	asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL,
	source_commit_id TEXT NOT NULL,
	selected_version_id INTEGER REFERENCES visual_asset_versions(id),
	history_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(history_json)),
	cursor INTEGER NOT NULL DEFAULT -1,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(asset_id,branch_id)
);
CREATE TABLE minigame_instances (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	turn INTEGER NOT NULL DEFAULT 0,
	protocol_version INTEGER NOT NULL,
	kind TEXT NOT NULL,
	phase TEXT NOT NULL CHECK(phase IN ('ready','active','paused','resolved')),
	instance_json TEXT NOT NULL CHECK(json_valid(instance_json)),
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE story_tts_settings (
	story_id TEXT PRIMARY KEY REFERENCES stories(id) ON DELETE CASCADE,
	mode TEXT NOT NULL DEFAULT 'off' CHECK(mode IN ('off','narrator','dialogue','all')),
	autoplay INTEGER NOT NULL DEFAULT 0 CHECK(autoplay IN (0,1)),
	default_language_tag TEXT NOT NULL DEFAULT '',
	provider_policy_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(provider_policy_json)),
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE voice_profiles (
	id TEXT PRIMARY KEY,
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	provider_voice_id TEXT NOT NULL,
	display_name TEXT NOT NULL,
	language_tags_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(language_tags_json)),
	traits_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(traits_json)),
	rights_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(rights_json)),
	version TEXT NOT NULL DEFAULT '',
	style_family TEXT NOT NULL DEFAULT 'neutral',
	enabled INTEGER NOT NULL DEFAULT 1 CHECK(enabled IN (0,1)),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(provider,model,provider_voice_id,version,style_family)
);
CREATE TABLE character_voice_assignments (
	id TEXT PRIMARY KEY,
	assignment_key TEXT NOT NULL,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	entity_id TEXT,
	identity_id TEXT,
	form_id TEXT,
	role TEXT NOT NULL CHECK(role IN ('narrator','protagonist','npc')),
	voice_profile_id TEXT NOT NULL REFERENCES voice_profiles(id),
	enabled_mode TEXT NOT NULL DEFAULT 'inherit' CHECK(enabled_mode IN ('inherit','on','off')),
	language_tag TEXT NOT NULL DEFAULT '',
	style_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(style_json)),
	locked INTEGER NOT NULL DEFAULT 0 CHECK(locked IN (0,1)),
	importance TEXT NOT NULL DEFAULT 'supporting' CHECK(importance IN ('major','supporting','minor')),
	allow_duplicate INTEGER NOT NULL DEFAULT 0 CHECK(allow_duplicate IN (0,1)),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id,assignment_key)
);
CREATE TABLE pronunciation_lexicon (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	language_tag TEXT NOT NULL,
	source_text TEXT NOT NULL,
	pronunciation TEXT NOT NULL,
	alphabet TEXT NOT NULL DEFAULT 'ipa' CHECK(alphabet IN ('ipa','x-sampa','provider')),
	case_sensitive INTEGER NOT NULL DEFAULT 0 CHECK(case_sensitive IN (0,1)),
	revision INTEGER NOT NULL DEFAULT 1 CHECK(revision > 0),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id,language_tag,source_text,case_sensitive)
);
CREATE TABLE tts_cache_entries (
	cache_key TEXT PRIMARY KEY,
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	provider_voice_id TEXT NOT NULL,
	voice_version TEXT NOT NULL DEFAULT '',
	language_tag TEXT NOT NULL,
	text_hash TEXT NOT NULL,
	style_hash TEXT NOT NULL,
	style_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(style_json)),
	speed REAL NOT NULL DEFAULT 1.0 CHECK(speed BETWEEN 0.25 AND 4.0),
	output_format TEXT NOT NULL CHECK(output_format IN ('mp3','opus','wav','aac','flac','pcm')),
	status TEXT NOT NULL CHECK(status IN ('pending','ready','failed','invalidated')),
	file_path TEXT NOT NULL DEFAULT '',
	duration_ms INTEGER NOT NULL DEFAULT 0 CHECK(duration_ms >= 0),
	timings_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(timings_json)),
	error TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE audio_assets (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	source_message_id INTEGER NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
	segment_index INTEGER NOT NULL CHECK(segment_index >= 0),
	segment_kind TEXT NOT NULL CHECK(segment_kind IN ('narrator','dialogue')),
	speaker_entity_id TEXT,
	identity_id TEXT,
	form_id TEXT,
	voice_profile_id TEXT NOT NULL REFERENCES voice_profiles(id),
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	provider_voice_id TEXT NOT NULL,
	voice_version TEXT NOT NULL DEFAULT '',
	language_tag TEXT NOT NULL,
	pronunciation_revision INTEGER NOT NULL DEFAULT 0 CHECK(pronunciation_revision >= 0),
	text TEXT NOT NULL,
	text_hash TEXT NOT NULL,
	cache_key TEXT NOT NULL,
	style_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(style_json)),
	speed REAL NOT NULL DEFAULT 1.0 CHECK(speed BETWEEN 0.25 AND 4.0),
	output_format TEXT NOT NULL CHECK(output_format IN ('mp3','opus','wav','aac','flac','pcm')),
	status TEXT NOT NULL CHECK(status IN ('pending','queued','running','ready','failed','cancelled','invalidated')),
	url TEXT NOT NULL DEFAULT '',
	file_path TEXT NOT NULL DEFAULT '',
	duration_ms INTEGER NOT NULL DEFAULT 0 CHECK(duration_ms >= 0),
	timings_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(timings_json)),
	generation_run_id TEXT REFERENCES generation_runs(id),
	error TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id,branch_id,source_message_id,segment_index,voice_profile_id,cache_key)
);
CREATE TABLE tts_jobs (
	id TEXT PRIMARY KEY,
	audio_asset_id TEXT NOT NULL UNIQUE REFERENCES audio_assets(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	status TEXT NOT NULL CHECK(status IN ('queued','running','succeeded','failed','cancelled')),
	provider TEXT NOT NULL,
	attempts INTEGER NOT NULL DEFAULT 0 CHECK(attempts >= 0),
	max_attempts INTEGER NOT NULL DEFAULT 3 CHECK(max_attempts BETWEEN 1 AND 10),
	next_attempt_at DATETIME,
	trace_id TEXT NOT NULL DEFAULT '',
	parent_run_id TEXT,
	generation_run_id TEXT REFERENCES generation_runs(id),
	error_class TEXT NOT NULL DEFAULT '',
	error TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
PRAGMA writable_schema=ON;
INSERT INTO sqlite_schema(type,name,tbl_name,rootpage,sql)VALUES('table','chat_messages_fts','chat_messages_fts',0,'CREATE VIRTUAL TABLE chat_messages_fts USING fts5(
	content,
	content=''chat_messages'',
	content_rowid=''id'',
	tokenize=''trigram''
)');
CREATE TABLE IF NOT EXISTS 'chat_messages_fts_data'(id INTEGER PRIMARY KEY, block BLOB);
INSERT INTO chat_messages_fts_data VALUES(1,X'0115');
INSERT INTO chat_messages_fts_data VALUES(10,X'000000000101010001010101');
INSERT INTO chat_messages_fts_data VALUES(137438953473,X'000000a3043020616e01021002026c690102050103616e73010211010365206101020f03016c010204020272730102150103676874010208010368652001020302026f7501020b02027468010209010369676801020701036c696701020601036e737701021201036f757301020c010372732e010216010373652001020e02027765010213010374686501020203016f01020a010375736501020d0103776572010214040807080806070808070708080808080807080608');
CREATE TABLE IF NOT EXISTS 'chat_messages_fts_idx'(segid, term, pgno, PRIMARY KEY(segid, term)) WITHOUT ROWID;
INSERT INTO chat_messages_fts_idx VALUES(1,X'',2);
CREATE TABLE IF NOT EXISTS 'chat_messages_fts_docsize'(id INTEGER PRIMARY KEY, sz BLOB);
INSERT INTO chat_messages_fts_docsize VALUES(1,X'15');
CREATE TABLE IF NOT EXISTS 'chat_messages_fts_config'(k PRIMARY KEY, v) WITHOUT ROWID;
INSERT INTO chat_messages_fts_config VALUES('version',4);
INSERT INTO sqlite_schema(type,name,tbl_name,rootpage,sql)VALUES('table','chapters_fts','chapters_fts',0,'CREATE VIRTUAL TABLE chapters_fts USING fts5(
	title,
	summary,
	content=''chapters'',
	content_rowid=''id'',
	tokenize=''trigram''
)');
CREATE TABLE IF NOT EXISTS 'chapters_fts_data'(id INTEGER PRIMARY KEY, block BLOB);
INSERT INTO chapters_fts_data VALUES(1,X'000000');
INSERT INTO chapters_fts_data VALUES(10,X'00000000000000');
CREATE TABLE IF NOT EXISTS 'chapters_fts_idx'(segid, term, pgno, PRIMARY KEY(segid, term)) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS 'chapters_fts_docsize'(id INTEGER PRIMARY KEY, sz BLOB);
CREATE TABLE IF NOT EXISTS 'chapters_fts_config'(k PRIMARY KEY, v) WITHOUT ROWID;
INSERT INTO chapters_fts_config VALUES('version',4);
CREATE TABLE image_masks (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
	source_version_id INTEGER NOT NULL REFERENCES visual_asset_versions(id),
	semantics TEXT NOT NULL CHECK(semantics='edit_coverage'),
	pixel_format TEXT NOT NULL CHECK(pixel_format='L8'),
	width INTEGER NOT NULL CHECK(width>0),
	height INTEGER NOT NULL CHECK(height>0),
	orientation INTEGER NOT NULL DEFAULT 1 CHECK(orientation=1),
	preserve_value INTEGER NOT NULL DEFAULT 0 CHECK(preserve_value=0),
	editable_value INTEGER NOT NULL DEFAULT 255 CHECK(editable_value=255),
	soft_edges INTEGER NOT NULL DEFAULT 0 CHECK(soft_edges IN (0,1)),
	mime_type TEXT NOT NULL CHECK(mime_type='image/png'),
	sha256 TEXT NOT NULL,
	file_path TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id,source_version_id,sha256)
);
CREATE TABLE image_operations (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
	operation TEXT NOT NULL CHECK(operation IN ('generate','edit','inpaint','image_transform','variation','reference_generate','outpaint')),
	status TEXT NOT NULL CHECK(status IN ('queued','running','succeeded','failed','cancelled')),
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	endpoint_id TEXT NOT NULL DEFAULT '',
	model_version TEXT NOT NULL DEFAULT '',
	deployment TEXT NOT NULL DEFAULT '',
	source_version_id INTEGER REFERENCES visual_asset_versions(id),
	parent_version_id INTEGER REFERENCES visual_asset_versions(id),
	mask_id TEXT REFERENCES image_masks(id),
	branch_id TEXT NOT NULL,
	source_commit_id TEXT NOT NULL,
	prompt TEXT NOT NULL,
	negative_prompt TEXT NOT NULL DEFAULT '',
	requested_parameters_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(requested_parameters_json)),
	effective_parameters_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(effective_parameters_json)),
	idempotency_key TEXT NOT NULL,
	provider_request_id TEXT NOT NULL DEFAULT '',
	result_version_id INTEGER REFERENCES visual_asset_versions(id),
	error_code TEXT NOT NULL DEFAULT '',
	error_summary TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	finished_at DATETIME,
	UNIQUE(story_id,idempotency_key)
);
CREATE TABLE provider_capability_snapshots (
	id TEXT PRIMARY KEY,
	provider TEXT NOT NULL,
	endpoint_id TEXT NOT NULL,
	model TEXT NOT NULL,
	model_version TEXT NOT NULL DEFAULT '',
	credential_mode TEXT NOT NULL,
	api_version TEXT NOT NULL DEFAULT '',
	schema_revision TEXT NOT NULL,
	capabilities_json TEXT NOT NULL CHECK(json_valid(capabilities_json)),
	provenance TEXT NOT NULL CHECK(provenance IN ('static_verified','provider_schema','runtime_probe')),
	schema_hash TEXT NOT NULL DEFAULT '',
	verified_at DATETIME NOT NULL,
	UNIQUE(provider,endpoint_id,model,model_version,api_version,schema_revision)
);
DELETE FROM sqlite_sequence;
INSERT INTO sqlite_sequence VALUES('chat_messages',1);
CREATE TRIGGER trg_turn_commits_immutable
BEFORE UPDATE ON turn_commits
WHEN NOT (
	OLD.payload_hash = '' AND NEW.payload_hash != ''
	AND OLD.id = NEW.id AND OLD.story_id = NEW.story_id AND OLD.branch_id = NEW.branch_id
	AND COALESCE(OLD.parent_commit_id,'') = COALESCE(NEW.parent_commit_id,'')
	AND OLD.canonical_turn = NEW.canonical_turn AND OLD.story_revision = NEW.story_revision
	AND OLD.kind = NEW.kind AND OLD.message = NEW.message AND OLD.created_at = NEW.created_at
)
BEGIN
	SELECT RAISE(ABORT, 'turn commits are immutable');
END;
CREATE TRIGGER trg_turn_snapshots_immutable
BEFORE UPDATE ON turn_snapshots
BEGIN
	SELECT RAISE(ABORT, 'turn snapshots are immutable');
END;
CREATE TRIGGER trg_canonical_events_immutable
BEFORE UPDATE ON canonical_events
BEGIN
	SELECT RAISE(ABORT, 'canonical events are immutable');
END;
CREATE TRIGGER trg_identity_claims_immutable BEFORE UPDATE ON identity_claims BEGIN SELECT RAISE(ABORT,'identity claims are append-only'); END;
CREATE TRIGGER trg_character_facts_immutable BEFORE UPDATE ON character_facts BEGIN SELECT RAISE(ABORT,'character facts are append-only'); END;
CREATE TRIGGER trg_controller_events_immutable BEFORE UPDATE ON entity_controller_events BEGIN SELECT RAISE(ABORT,'controller events are append-only'); END;
CREATE TRIGGER trg_lifecycle_events_immutable BEFORE UPDATE ON entity_lifecycle_events BEGIN SELECT RAISE(ABORT,'lifecycle events are append-only'); END;
CREATE TRIGGER trg_reputation_events_immutable BEFORE UPDATE ON reputation_events BEGIN SELECT RAISE(ABORT,'reputation events are append-only'); END;
CREATE TRIGGER trg_membership_events_immutable BEFORE UPDATE ON faction_membership_events BEGIN SELECT RAISE(ABORT,'membership events are append-only'); END;
CREATE TRIGGER trg_faction_relationship_events_immutable BEFORE UPDATE ON faction_relationship_events BEGIN SELECT RAISE(ABORT,'faction relationship events are append-only'); END;
CREATE TRIGGER trg_position_events_immutable BEFORE UPDATE ON entity_position_events BEGIN SELECT RAISE(ABORT,'position events are append-only'); END;
CREATE TRIGGER trg_world_time_events_immutable BEFORE UPDATE ON world_time_events BEGIN SELECT RAISE(ABORT,'world time events are append-only'); END;
CREATE TRIGGER trg_weather_states_immutable BEFORE UPDATE ON weather_states BEGIN SELECT RAISE(ABORT,'weather states are append-only'); END;
CREATE TRIGGER trg_canonical_world_events_immutable BEFORE UPDATE ON canonical_world_events BEGIN SELECT RAISE(ABORT,'world events are append-only'); END;
CREATE TRIGGER trg_challenge_runs_immutable
BEFORE UPDATE ON challenge_runs
WHEN NOT (OLD.source_commit_id='' AND NEW.source_commit_id!='' AND OLD.branch_id=NEW.branch_id)
BEGIN SELECT RAISE(ABORT,'challenge runs are immutable'); END;
CREATE TRIGGER trg_prompt_revisions_immutable BEFORE UPDATE ON prompt_profile_revisions
BEGIN SELECT RAISE(ABORT,'prompt profile revisions are immutable'); END;
CREATE TRIGGER trg_generation_events_immutable BEFORE UPDATE ON generation_events
BEGIN SELECT RAISE(ABORT,'generation events are append-only'); END;
CREATE TRIGGER trg_minigame_instance_lineage_immutable
BEFORE UPDATE ON minigame_instances
WHEN OLD.story_id!=NEW.story_id OR OLD.branch_id!=NEW.branch_id OR OLD.source_commit_id!=NEW.source_commit_id OR OLD.kind!=NEW.kind OR OLD.protocol_version!=NEW.protocol_version
BEGIN SELECT RAISE(ABORT,'minigame lineage is immutable'); END;
CREATE TRIGGER trg_audio_asset_lineage_immutable
BEFORE UPDATE ON audio_assets
WHEN OLD.story_id!=NEW.story_id OR OLD.branch_id!=NEW.branch_id OR OLD.source_commit_id!=NEW.source_commit_id OR OLD.source_message_id!=NEW.source_message_id OR OLD.segment_index!=NEW.segment_index
BEGIN SELECT RAISE(ABORT,'audio asset lineage is immutable'); END;
CREATE TRIGGER trg_tts_job_lineage_immutable
BEFORE UPDATE ON tts_jobs
WHEN OLD.story_id!=NEW.story_id OR OLD.branch_id!=NEW.branch_id OR OLD.source_commit_id!=NEW.source_commit_id OR OLD.audio_asset_id!=NEW.audio_asset_id
BEGIN SELECT RAISE(ABORT,'tts job lineage is immutable'); END;
CREATE TRIGGER trg_chat_messages_fts_insert AFTER INSERT ON chat_messages BEGIN
	INSERT INTO chat_messages_fts(rowid,content) VALUES (new.id,new.content);
END;
CREATE TRIGGER trg_chat_messages_fts_delete AFTER DELETE ON chat_messages BEGIN
	INSERT INTO chat_messages_fts(chat_messages_fts,rowid,content) VALUES ('delete',old.id,old.content);
END;
CREATE TRIGGER trg_chat_messages_fts_update AFTER UPDATE OF content ON chat_messages BEGIN
	INSERT INTO chat_messages_fts(chat_messages_fts,rowid,content) VALUES ('delete',old.id,old.content);
	INSERT INTO chat_messages_fts(rowid,content) VALUES (new.id,new.content);
END;
CREATE TRIGGER trg_chapters_fts_insert AFTER INSERT ON chapters BEGIN
	INSERT INTO chapters_fts(rowid,title,summary) VALUES (new.id,new.title,new.summary);
END;
CREATE TRIGGER trg_chapters_fts_delete AFTER DELETE ON chapters BEGIN
	INSERT INTO chapters_fts(chapters_fts,rowid,title,summary) VALUES ('delete',old.id,old.title,old.summary);
END;
CREATE TRIGGER trg_chapters_fts_update AFTER UPDATE OF title,summary ON chapters BEGIN
	INSERT INTO chapters_fts(chapters_fts,rowid,title,summary) VALUES ('delete',old.id,old.title,old.summary);
	INSERT INTO chapters_fts(rowid,title,summary) VALUES (new.id,new.title,new.summary);
END;
CREATE INDEX idx_characters_story ON characters(story_id);
CREATE INDEX idx_npcs_story ON npcs(story_id);
CREATE INDEX idx_sessions_story ON sessions(story_id);
CREATE INDEX idx_chapters_story ON chapters(story_id);
CREATE INDEX idx_achievements_story ON achievements(story_id);
CREATE INDEX idx_saves_story ON saves(story_id);
CREATE INDEX idx_rag_chunks_story ON rag_chunks(story_id);
CREATE INDEX idx_rag_chunks_story_type ON rag_chunks(story_id, chunk_type);
CREATE INDEX idx_combat_log_story ON combat_log(story_id);
CREATE INDEX idx_chat_messages_session ON chat_messages(session_id);
CREATE INDEX idx_chat_messages_story ON chat_messages(story_id);
CREATE INDEX idx_chat_messages_session_turn_id
	ON chat_messages(session_id, turn, id);
CREATE INDEX idx_chat_messages_story_turn_id
	ON chat_messages(story_id, turn, id);
CREATE INDEX idx_npcs_story_last_seen
	ON npcs(story_id, last_seen_turn DESC);
CREATE INDEX idx_achievements_story_name_ci
	ON achievements(story_id, name COLLATE NOCASE);
CREATE INDEX idx_turn_idempotency_created_at
	ON turn_idempotency(created_at);
CREATE INDEX idx_story_turn_locks_locked_until
	ON story_turn_locks(locked_until);
CREATE INDEX idx_turn_idempotency_status_locked_until
		ON turn_idempotency(status, locked_until);
CREATE INDEX idx_turn_idempotency_request_hash
		ON turn_idempotency(story_id, idempotency_key, request_hash);
CREATE INDEX idx_visual_asset_versions_asset
	ON visual_asset_versions(asset_id, created_at DESC);
CREATE INDEX idx_visual_asset_versions_story
	ON visual_asset_versions(story_id, kind, subject, created_at DESC);
CREATE INDEX idx_visual_generation_jobs_status_lock
	ON visual_generation_jobs(status, locked_until, created_at);
CREATE INDEX idx_visual_generation_jobs_story
	ON visual_generation_jobs(story_id, status, created_at);
CREATE INDEX idx_npcs_story_last_seen_alive
			ON npcs(story_id, is_alive, last_seen_turn DESC);
CREATE INDEX idx_story_branches_story ON story_branches(story_id, created_at);
CREATE INDEX idx_turn_commits_story_branch_turn ON turn_commits(story_id, branch_id, canonical_turn, created_at);
CREATE INDEX idx_turn_commits_parent ON turn_commits(parent_commit_id, created_at);
CREATE INDEX idx_turn_snapshots_story ON turn_snapshots(story_id, created_at);
CREATE INDEX idx_canonical_events_commit ON canonical_events(commit_id, sequence);
CREATE INDEX idx_save_bookmarks_story ON save_bookmarks(story_id, created_at DESC);
CREATE INDEX idx_generation_traces_lineage ON generation_traces(story_id, branch_id, source_commit_id, created_at);
CREATE INDEX idx_audio_artifacts_lineage ON audio_artifacts(story_id, branch_id, source_commit_id, created_at);
CREATE INDEX idx_chat_messages_branch_turn ON chat_messages(story_id, branch_id, turn, id);
CREATE INDEX idx_chapters_branch_number ON chapters(story_id, branch_id, chapter_number);
CREATE INDEX idx_rag_chunks_branch_turn ON rag_chunks(story_id, branch_id, turn_end);
CREATE INDEX idx_saves_branch_created ON saves(story_id, branch_id, created_at DESC);
CREATE INDEX idx_identity_claims_observer ON identity_claims(story_id,subject_entity_id,observer_entity_id,status,learned_turn);
CREATE INDEX idx_character_facts_projection ON character_facts(story_id,subject_entity_id,observer_entity_id,visibility,learned_turn);
CREATE INDEX idx_entity_forms_history ON entity_forms(story_id,entity_id,valid_from_turn);
CREATE INDEX idx_reputation_ledger ON reputation_events(story_id,faction_id,entity_id,turn);
CREATE UNIQUE INDEX idx_npcs_canonical_entity ON npcs(story_id,canonical_entity_id) WHERE canonical_entity_id!='';
CREATE INDEX idx_locations_story_discovery ON locations(story_id,discovery_state,canonical_name);
CREATE INDEX idx_location_edges_from ON location_edges(story_id,from_location_id,visibility);
CREATE INDEX idx_world_time_events_story ON world_time_events(story_id,created_at);
CREATE INDEX idx_weather_current ON weather_states(story_id,location_id,valid_from_day,valid_from_minute);
CREATE INDEX idx_challenge_runs_lineage ON challenge_runs(story_id,branch_id,source_commit_id,turn,created_at);
CREATE INDEX idx_generation_runs_trace ON generation_runs(trace_id, created_at, id);
CREATE INDEX idx_generation_runs_story_lineage ON generation_runs(story_id, branch_id, source_commit_id, created_at);
CREATE INDEX idx_generation_runs_message ON generation_runs(message_id, created_at);
CREATE INDEX idx_generation_attempts_run ON generation_attempts(run_id, sequence);
CREATE INDEX idx_generation_events_run ON generation_events(run_id, id);
CREATE INDEX idx_prompt_revisions_profile ON prompt_profile_revisions(profile_id, version DESC);
CREATE INDEX idx_visual_assets_story_kind ON visual_assets(story_id,kind,updated_at DESC);
CREATE INDEX idx_visual_assets_branch_kind ON visual_assets(story_id,branch_id,kind,updated_at DESC);
CREATE INDEX idx_visual_assets_lineage ON visual_assets(story_id,lineage_key,branch_id,source_commit_id);
CREATE INDEX idx_visual_profile_revisions_reachable ON visual_profile_revisions(story_id,source_commit_id,revision DESC);
CREATE INDEX idx_visual_asset_versions_lineage ON visual_asset_versions(story_id,branch_id,source_commit_id,appearance_fingerprint,id DESC);
CREATE INDEX idx_visual_jobs_lineage ON visual_generation_jobs(story_id,branch_id,source_commit_id,status,created_at);
CREATE INDEX idx_visual_overrides_lineage ON visual_asset_branch_overrides(story_id,asset_id,source_commit_id);
CREATE UNIQUE INDEX idx_visual_generation_jobs_active_asset ON visual_generation_jobs(asset_id,branch_id) WHERE status IN ('queued','running');
CREATE INDEX idx_minigame_instances_branch_phase
	ON minigame_instances(story_id,branch_id,phase,updated_at DESC);
CREATE UNIQUE INDEX idx_major_voice_unique
	ON character_voice_assignments(story_id,voice_profile_id)
	WHERE importance='major' AND allow_duplicate=0 AND enabled_mode!='off';
CREATE INDEX idx_voice_assignments_story
	ON character_voice_assignments(story_id,role,entity_id,form_id);
CREATE INDEX idx_tts_cache_identity
	ON tts_cache_entries(provider,model,provider_voice_id,voice_version,language_tag,text_hash,style_hash,speed,output_format);
CREATE INDEX idx_audio_assets_lineage_v35
	ON audio_assets(story_id,branch_id,source_commit_id,source_message_id,segment_index);
CREATE INDEX idx_audio_assets_cache
	ON audio_assets(cache_key,status);
CREATE INDEX idx_tts_jobs_queue
	ON tts_jobs(status,next_attempt_at,created_at);
CREATE INDEX idx_tts_jobs_lineage
	ON tts_jobs(story_id,branch_id,source_commit_id,created_at);
CREATE INDEX idx_regions_story_parent
	ON regions(story_id,parent_region_id,visibility,name);
CREATE INDEX idx_locations_story_scope
	ON locations(story_id,region_id,parent_location_id,discovery_state,canonical_name);
CREATE UNIQUE INDEX idx_location_edges_canonical_route
	ON location_edges(story_id,branch_id,from_location_id,to_location_id,direction,travel_mode);
CREATE INDEX idx_visual_assets_map_scope
	ON visual_assets(story_id,branch_id,kind,map_scope_kind,map_scope_id,updated_at DESC);
CREATE INDEX idx_character_facts_visible
	ON character_facts(story_id,branch_id,subject_entity_id,visibility,learned_turn)
	WHERE retracts_fact_id IS NULL;
CREATE INDEX idx_character_facts_retracts
	ON character_facts(story_id,branch_id,retracts_fact_id)
	WHERE retracts_fact_id IS NOT NULL;
CREATE INDEX idx_character_facts_supersedes
	ON character_facts(story_id,branch_id,supersedes_fact_id)
	WHERE supersedes_fact_id IS NOT NULL;
CREATE INDEX idx_turn_idempotency_retention
	ON turn_idempotency(story_id,status,updated_at DESC);
CREATE INDEX idx_image_operations_queue ON image_operations(status,created_at);
CREATE INDEX idx_image_operations_asset ON image_operations(story_id,asset_id,created_at DESC);
CREATE INDEX idx_image_masks_source ON image_masks(story_id,source_version_id);
CREATE INDEX idx_visual_versions_parent ON visual_asset_versions(asset_id,parent_version_id,id DESC);
PRAGMA writable_schema=OFF;
COMMIT;

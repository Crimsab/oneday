import { describe,expect,it } from "vitest";
import { dialogueBlocksFromMessage } from "./Transcript";
import type { MessageView } from "../types";

describe("structured dialogue",()=>{
	it("uses speaker ids and drops malformed empty blocks",()=>{
		const message:MessageView={id:1,session_id:"s",story_id:"story",turn:2,role:"assistant",content:"A pause.",message_type:"narrative",created_at:"now",branch_id:"main",source_commit_id:"commit",metadata:{output:{dialogue_blocks:[{speaker_id:"entity-mara",speaker:"Mara",role:"npc",text:" Wait. "},{speaker:"Nobody",text:""}]}}};
		expect(dialogueBlocksFromMessage(message)).toEqual([{speakerId:"entity-mara",speaker:"Mara",role:"npc",text:"Wait."}]);
	});
});

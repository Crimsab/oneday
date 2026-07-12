import { describe,expect,it } from "vitest";
import { dialogueBlocksFromMessage, isTranscriptNearBottom, latestAutoplayMessageId } from "./Transcript";
import type { MessageView } from "../types";

describe("structured dialogue",()=>{
	it("uses speaker ids and drops malformed empty blocks",()=>{
		const message:MessageView={id:1,session_id:"s",story_id:"story",turn:2,role:"assistant",content:"A pause.",message_type:"narrative",created_at:"now",branch_id:"main",source_commit_id:"commit",metadata:{output:{dialogue_blocks:[{speaker_id:"entity-mara",speaker:"Mara",role:"npc",text:" Wait. "},{speaker:"Nobody",text:""}]}}};
		expect(dialogueBlocksFromMessage(message)).toEqual([{speakerId:"entity-mara",speaker:"Mara",role:"npc",text:"Wait."}]);
	});
});

describe("transcript follow mode", () => {
	it("follows only while the reader remains near the newest message", () => {
		expect(isTranscriptNearBottom({ scrollTop: 910, scrollHeight: 1500, clientHeight: 500 })).toBe(true);
		expect(isTranscriptNearBottom({ scrollTop: 700, scrollHeight: 1500, clientHeight: 500 })).toBe(false);
		expect(isTranscriptNearBottom({ scrollTop: 904, scrollHeight: 1500, clientHeight: 500 }, 95)).toBe(false);
	});
});

describe("transcript audio ownership", () => {
	it("assigns autoplay only to the newest canonical narrator message", () => {
		const messages = [
			{ id: 1, role: "assistant", source_commit_id: "commit-1" },
			{ id: 2, role: "user", source_commit_id: "commit-2" },
			{ id: 3, role: "assistant" },
			{ id: 4, role: "assistant", source_commit_id: "commit-4" },
		] as MessageView[];
		expect(latestAutoplayMessageId(messages)).toBe(4);
	});
});

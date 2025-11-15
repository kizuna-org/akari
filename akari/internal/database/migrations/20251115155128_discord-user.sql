-- Create "discord_users" table
CREATE TABLE "discord_users" (
  "id" character varying NOT NULL,
  "username" character varying NOT NULL,
  "bot" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);
-- Drop index "discordmessage_author_id_timestamp" from table: "discord_messages"
DROP INDEX "discordmessage_author_id_timestamp";
-- Rename a column from "author_id" to "discord_message_author"
ALTER TABLE "discord_messages" RENAME COLUMN "author_id" TO "discord_message_author";
-- Modify "discord_messages" table
ALTER TABLE "discord_messages" ADD CONSTRAINT "discord_messages_discord_users_author" FOREIGN KEY ("discord_message_author") REFERENCES "discord_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;

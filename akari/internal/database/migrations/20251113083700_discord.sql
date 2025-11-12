-- Create "discord_guilds" table
CREATE TABLE "discord_guilds" (
  "id" character varying NOT NULL,
  "name" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);
-- Create "discord_channels" table
CREATE TABLE "discord_channels" (
  "id" character varying NOT NULL,
  "name" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "discord_channel_guild" character varying NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "discord_channels_discord_guilds_guild" FOREIGN KEY ("discord_channel_guild") REFERENCES "discord_guilds" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create "discord_messages" table
CREATE TABLE "discord_messages" (
  "id" character varying NOT NULL,
  "author_id" character varying NOT NULL,
  "content" character varying NOT NULL,
  "timestamp" timestamptz NOT NULL,
  "mentions" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "discord_message_channel" character varying NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "discord_messages_discord_channels_channel" FOREIGN KEY ("discord_message_channel") REFERENCES "discord_channels" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "discordmessage_author_id_timestamp" to table: "discord_messages"
CREATE INDEX "discordmessage_author_id_timestamp" ON "discord_messages" ("author_id", "timestamp");

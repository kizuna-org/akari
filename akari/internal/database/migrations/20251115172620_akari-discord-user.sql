-- Modify "discord_users" table
ALTER TABLE "discord_users" ADD COLUMN "akari_user_discord_user" bigint NOT NULL, ADD CONSTRAINT "discord_users_akari_users_discord_user" FOREIGN KEY ("akari_user_discord_user") REFERENCES "akari_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Create index "discord_users_akari_user_discord_user_key" to table: "discord_users"
CREATE UNIQUE INDEX "discord_users_akari_user_discord_user_key" ON "discord_users" ("akari_user_discord_user");

UPDATE collections
SET invite_token = translate(encode(uuid_send(gen_random_uuid()), 'base64'), '+/=', '-_')
WHERE invite_token IS NULL OR invite_token = '';

ALTER TABLE collections
    ALTER COLUMN invite_token SET NOT NULL;

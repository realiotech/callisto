
/* ---- REWARD EARNED ---- */

CREATE TABLE reward_earned
(
    delegator_address TEXT NOT NULL,
    denom             TEXT NOT NULL,
    amount            TEXT NOT NULL,
    height            BIGINT NOT NULL,
    PRIMARY KEY (delegator_address)
);
CREATE INDEX reward_earned_delegator_address_index ON reward_earned (delegator_address);
CREATE INDEX reward_earned_height_index ON reward_earned (height);

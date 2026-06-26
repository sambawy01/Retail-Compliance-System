-- 008_lawful_basis.sql
-- Watch Dog retail compliance system — add lawful_basis to consent records per PDP Law.

-- PDP Law (Egypt's Personal Data Protection Law) requires recording the
-- lawful basis for processing biometric data. Common bases:
-- - consent: explicit, revocable consent from the data subject
-- - legitimate_interest: necessary for a legitimate business purpose
-- - legal_obligation: required by law or regulation
-- For biometric data in a retail compliance context, "consent" is the
-- primary basis, but other bases may apply for specific use cases.

ALTER TABLE identity_consents
ADD COLUMN lawful_basis TEXT NOT NULL DEFAULT 'consent'
    CHECK (lawful_basis IN ('consent', 'legitimate_interest', 'legal_obligation'));

-- Add a comment for documentation
COMMENT ON COLUMN identity_consents.lawful_basis IS
    'PDP Law: lawful basis for processing biometric data (consent | legitimate_interest | legal_obligation)';

UPDATE suppliers
SET auto_approve_groups = COALESCE((
    SELECT jsonb_object_agg(platform, CASE jsonb_typeof(group_ids)
        WHEN 'number' THEN jsonb_build_array(group_ids)
        ELSE group_ids
    END)
    FROM jsonb_each(auto_approve_groups) AS groups(platform, group_ids)
), '{}'::jsonb)
WHERE EXISTS (
    SELECT 1 FROM jsonb_each(auto_approve_groups) AS groups(platform, group_ids)
    WHERE jsonb_typeof(group_ids) = 'number'
);

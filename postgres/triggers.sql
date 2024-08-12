CREATE OR REPLACE FUNCTION notify_postgraphile_trigger() RETURNS trigger AS $$
BEGIN
  IF TG_ARGV[0] IS NULL OR TG_ARGV[0] = '' THEN
    RAISE EXCEPTION 'Trigger argument cannot be null or empty';
  END IF;
  PERFORM pg_notify('postgraphile:' || TG_ARGV[0], '{}');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER notify_postgraphile_trigger
AFTER INSERT OR UPDATE OR DELETE ON public.inputs
FOR EACH STATEMENT EXECUTE FUNCTION notify_postgraphile_trigger("inputs");

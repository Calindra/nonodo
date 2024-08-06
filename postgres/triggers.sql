CREATE OR REPLACE FUNCTION notify_postgraphile_trigger() RETURNS trigger AS $$
BEGIN
  PERFORM pg_notify('postgraphile:' || TG_ARGV[0], '{}');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER notify_postgraphile_trigger
AFTER INSERT OR UPDATE OR DELETE ON public.inputs
FOR EACH STATEMENT EXECUTE FUNCTION notify_postgraphile_trigger("inputs");

CREATE OR REPLACE FUNCTION notify_trigger() RETURNS trigger AS $$
BEGIN
  PERFORM pg_notify('postgraphile:inputs', '{}');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER notify_trigger
AFTER INSERT OR UPDATE OR DELETE ON public.inputs
FOR EACH ROW EXECUTE FUNCTION notify_trigger();

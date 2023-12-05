namespace Datadog.CustomActions.Interfaces
{
    public interface IDecompressionStrategy
    {
        void Decompress(ISession session, string compressedFileName);
    }
}

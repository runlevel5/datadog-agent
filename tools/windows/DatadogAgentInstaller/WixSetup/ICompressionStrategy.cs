using System.IO;

namespace WixSetup
{
    public interface ICompressionStrategy
    {
        void Compress(FileInfo[] files, string archiveName, string sourceDir);
    }
}

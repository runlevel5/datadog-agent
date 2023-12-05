using System.IO;
using System.Text;
using Datadog.CustomActions.Interfaces;
using ICSharpCode.SharpZipLib.Tar;

namespace Datadog.CustomActions
{
    public class ManagedTarGzDecompressionStrategy : IDecompressionStrategy
    {
        public void Decompress(ISession session, string compressedFileName)
        {
            var decoder = new SevenZip.Compression.LZMA.Decoder();
            using (var inStream = File.OpenRead(compressedFileName))
            {
                using (var outStream = File.Create($"{compressedFileName}.tar"))
                {
                    var reader = new BinaryReader(inStream, Encoding.UTF8);
                    // Properties of the stream are encoded on 5 bytes
                    var props = reader.ReadBytes(5);
                    decoder.SetDecoderProperties(props);
                    var length = reader.ReadInt64();
                    decoder.Code(inStream, outStream, inStream.Length, length, null);
                    outStream.Flush();
                }
            }
            var outputPath = Path.GetDirectoryName(Path.GetFullPath(compressedFileName));
            using (var inStream = File.OpenRead($"{compressedFileName}.tar"))
            using (var tarArchive = TarArchive.CreateInputTarArchive(inStream, Encoding.UTF8))
            {
                tarArchive.ExtractContents(outputPath);
            }
            File.Delete($"{compressedFileName}.tar");
            File.Delete($"{compressedFileName}");
        }
    }
}

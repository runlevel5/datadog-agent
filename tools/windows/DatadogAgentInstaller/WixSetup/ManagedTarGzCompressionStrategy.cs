using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;
using ICSharpCode.SharpZipLib.Tar;
using SevenZip;

namespace WixSetup
{
    public class ManagedTarGzCompressionStrategy : ICompressionStrategy
    {
        public void Compress(FileInfo[] files, string archiveName, string sourceDir)
        {
            var tar = $"{archiveName}.tar";
            var sourceDirName = Path.GetFileName(sourceDir);
            using (var outStream = File.Create(tar))
            {
                using var tarArchive = new TarOutputStream(outStream, Encoding.UTF8);
                foreach (var file in files)
                {
                    // Path in tar must be in UNIX format
                    var nameInTar = $"{sourceDirName}{file.FullName.Substring(sourceDir.Length)}".Replace('\\', '/');
                    var entry = TarEntry.CreateTarEntry(nameInTar);
                    using var fileStream = File.OpenRead(file.FullName);
                    entry.Size = fileStream.Length;
                    tarArchive.PutNextEntry(entry);
                    fileStream.CopyTo(tarArchive);
                    tarArchive.CloseEntry();
                }
            }

            using (var inStream = File.Open(tar, FileMode.Open))
            using (var outStream = File.Create(archiveName))
            {
                Compress(inStream, outStream);
            }
            File.Delete(tar);
        }

        static void Compress(Stream inStream, Stream outStream)
        {
            var encoder = new SevenZip.Compression.LZMA.Encoder();
            var encodingProps = new Dictionary<CoderPropID, object>
            {
                {CoderPropID.DictionarySize, 32 * 1024 * 1024},
                {CoderPropID.PosStateBits,   2},
                {CoderPropID.LitContextBits, 3},
                {CoderPropID.LitPosBits,     0},
                {CoderPropID.Algorithm,      2},
                {CoderPropID.NumFastBytes,   64},
                {CoderPropID.MatchFinder,    "bt4"}
            };

            encoder.SetCoderProperties(encodingProps.Keys.ToArray(), encodingProps.Values.ToArray());
            encoder.WriteCoderProperties(outStream);
            var writer = new BinaryWriter(outStream, Encoding.UTF8);
            writer.Write(inStream.Length - inStream.Position);
            encoder.Code(inStream, outStream, -1, -1, null);
        }
    }
}

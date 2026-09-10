package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

// ── GetSuffix ──

func TestGetSuffix_Standard(t *testing.T) {
	assert.Equal(t, "mp4", GetSuffix("video.mp4"))
	assert.Equal(t, "avi", GetSuffix("movie.avi"))
	assert.Equal(t, "jpg", GetSuffix("photo.jpg"))
}

func TestGetSuffix_NoExt(t *testing.T) {
	assert.Empty(t, GetSuffix("README"))
}

func TestGetSuffix_Empty(t *testing.T) {
	assert.Empty(t, GetSuffix(""))
}

func TestGetSuffix_Lowercase(t *testing.T) {
	assert.Equal(t, "mp4", GetSuffix("video.MP4"))
}

// ── GetTitle ──

func TestGetTitle_RemovesExt(t *testing.T) {
	assert.Equal(t, "video", GetTitle("video.mp4"))
}

func TestGetTitle_NoExt(t *testing.T) {
	assert.Equal(t, "README", GetTitle("README"))
}

func TestGetTitle_Empty(t *testing.T) {
	assert.Empty(t, GetTitle(""))
}

func TestGetTitle_MultipleDots(t *testing.T) {
	assert.Equal(t, "archive.tar", GetTitle("archive.tar.gz"))
}

// ── GetMovieType ──

func TestGetMovieType_WithType(t *testing.T) {
	assert.Equal(t, "动漫", GetMovieType("video{{动漫}}.mp4"))
	assert.Equal(t, "电影", GetMovieType("movie{{电影}}.avi"))
}

func TestGetMovieType_NoType(t *testing.T) {
	assert.Equal(t, "无", GetMovieType("video.mp4"))
}

func TestGetMovieType_Empty(t *testing.T) {
	assert.Equal(t, "无", GetMovieType(""))
}

// ── GetCode ──

func TestGetCode_WithCode(t *testing.T) {
	assert.Equal(t, "ABC-123", GetCode("video [ABC-123].mp4"))
}

func TestGetCode_WithUnderscore(t *testing.T) {
	assert.Equal(t, "ABC_123", GetCode("video [ABC_123].mp4"))
}

func TestGetCode_NoBrackets(t *testing.T) {
	code := GetCode("video.mp4")
	assert.NotEmpty(t, code, "should fallback to title")
}

func TestGetCode_Empty(t *testing.T) {
	assert.Empty(t, GetCode(""))
}

// ── GetAuthor ──

func TestGetAuthor_WithBrackets(t *testing.T) {
	author := GetAuthor("movie [导演名].mp4")
	assert.Equal(t, "导演名", author)
}

func TestGetAuthor_NoBracketsLong(t *testing.T) {
	author := GetAuthor("this is a very long title without brackets.mp4")
	assert.Equal(t, 20, len(author))
}

func TestGetAuthor_NoBracketsShort(t *testing.T) {
	author := GetAuthor("short.mp4")
	assert.Equal(t, "short", author)
}

func TestGetAuthor_NoBracketsLongChinese(t *testing.T) {
	// 30 个汉字，超出 20 个字符，应按 rune 截断且不产生乱码
	author := GetAuthor("这是一个非常非常长的中文标题一共三十个汉字用来测试截断.mp4")
	assert.Equal(t, 20, utf8.RuneCountInString(author))
	assert.True(t, utf8.ValidString(author))
	assert.Equal(t, "这是一个非常非常长的中文标题一共三十个汉", author)
}

func TestGetAuthor_Empty(t *testing.T) {
	assert.Empty(t, GetAuthor(""))
}

// ── TruncateRunes / TailBytes ──

func TestTruncateRunes_RuneBoundary(t *testing.T) {
	assert.Equal(t, "这是一个非常非常长的中文标题一共三十个汉", TruncateRunes("这是一个非常非常长的中文标题一共三十个汉字用来测试截断", 20))
	assert.Equal(t, "abc", TruncateRunes("abc", 20))
	assert.Equal(t, "ab", TruncateRunes("abcdef", 2))
	assert.Empty(t, TruncateRunes("abc", 0))
	assert.Empty(t, TruncateRunes("abc", -1))
}

func TestTruncateRunes_ValidUTF8(t *testing.T) {
	out := TruncateRunes("中文标题很长很长很长很长很长很长", 5)
	assert.Equal(t, "中文标题很", out)
	assert.True(t, utf8.ValidString(out))
}

func TestTailBytes_NoMojibake(t *testing.T) {
	// 末尾 10 字节切在汉字中间时，应向前回退到合法字符边界
	s := "日志路径: 中文测试目录"
	out := TailBytes(s, 10)
	assert.True(t, utf8.ValidString(out))
	assert.True(t, strings.HasSuffix(s, out))
	assert.LessOrEqual(t, len(out), 10)
}

func TestTailBytes_ShortAndEmpty(t *testing.T) {
	assert.Equal(t, "abc", TailBytes("abc", 10))
	assert.Empty(t, TailBytes("abc", 0))
	assert.Equal(t, "日志", TailBytes("日志", 6))
}

// ── GetTags ──

func TestGetTags_WithTags(t *testing.T) {
	tags := GetTags("video《action,drama》.mp4", "")
	assert.Equal(t, []string{"action", "drama"}, tags)
}

func TestGetTags_NoTags(t *testing.T) {
	assert.Nil(t, GetTags("video.mp4", ""))
}

func TestGetTags_WithMovieType(t *testing.T) {
	tags := GetTags("video《action》.mp4", "动漫")
	assert.Contains(t, tags, "动漫")
	assert.Contains(t, tags, "action")
}

func TestGetTags_Empty(t *testing.T) {
	assert.Nil(t, GetTags("", ""))
}

// ── GetTagStr ──

func TestGetTagStr_WithTags(t *testing.T) {
	assert.Equal(t, "action,drama", GetTagStr("video《action,drama》.mp4"))
}

func TestGetTagStr_NoTags(t *testing.T) {
	assert.Empty(t, GetTagStr("video.mp4"))
}

func TestGetTagStr_Empty(t *testing.T) {
	assert.Empty(t, GetTagStr(""))
}

// ── GetSeriesByCode ──

func TestGetSeriesByCode_WithDash(t *testing.T) {
	assert.Equal(t, "ABC", GetSeriesByCode("ABC-123"))
}

func TestGetSeriesByCode_NoDash(t *testing.T) {
	assert.Empty(t, GetSeriesByCode("ABC123"))
}

func TestGetSeriesByCode_Empty(t *testing.T) {
	assert.Empty(t, GetSeriesByCode(""))
}

// ── GetSizeStr ──

func TestGetSizeStr_Bytes(t *testing.T) {
	assert.Equal(t, "100", GetSizeStr(100))
}

func TestGetSizeStr_KiloBytes(t *testing.T) {
	assert.Equal(t, "1 k", GetSizeStr(1500))
}

func TestGetSizeStr_MegaBytes(t *testing.T) {
	assert.Equal(t, "100.00 M", GetSizeStr(100*1024*1024))
}

func TestGetSizeStr_GigaBytes(t *testing.T) {
	assert.Equal(t, "2.00 G", GetSizeStr(2*1024*1024*1024))
}

func TestGetSizeStr_TeraBytes(t *testing.T) {
	assert.Equal(t, "2.00 T", GetSizeStr(2*1024*1024*1024*1024))
}

func TestGetSizeStr_Zero(t *testing.T) {
	assert.Equal(t, "0", GetSizeStr(0))
}

// ── DirpathForId ──

func TestDirpathForId_Deterministic(t *testing.T) {
	id1 := DirpathForId("/path/to/file.mp4")
	id2 := DirpathForId("/path/to/file.mp4")
	assert.Equal(t, id1, id2)
}

func TestDirpathForId_Different(t *testing.T) {
	id1 := DirpathForId("/path/a.mp4")
	id2 := DirpathForId("/path/b.mp4")
	assert.NotEqual(t, id1, id2)
}

func TestDirpathForId_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, DirpathForId(""))
}

// ── ExistsFiles ──

func TestExistsFiles_Exists(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmpFile, []byte("hello"), 0644)
	assert.True(t, ExistsFiles(tmpFile))
}

func TestExistsFiles_NotExists(t *testing.T) {
	assert.False(t, ExistsFiles("/nonexistent/path/file.txt"))
}

func TestExistsFiles_Empty(t *testing.T) {
	assert.False(t, ExistsFiles(""))
}

// ── ValidatePath ──

func TestValidatePath_Allowed(t *testing.T) {
	basedir := t.TempDir()
	subdir := filepath.Join(basedir, "sub")
	os.Mkdir(subdir, 0755)

	abs, err := ValidatePath(filepath.Join(basedir, "test.mp4"), []string{basedir})
	assert.NoError(t, err)
	assert.NotEmpty(t, abs)
}

func TestValidatePath_SubDir(t *testing.T) {
	basedir := t.TempDir()
	subdir := filepath.Join(basedir, "sub")
	os.Mkdir(subdir, 0755)
	os.WriteFile(filepath.Join(subdir, "test.mp4"), []byte("data"), 0644)

	abs, err := ValidatePath(filepath.Join(subdir, "test.mp4"), []string{basedir})
	assert.NoError(t, err)
	assert.Contains(t, abs, "test.mp4")
}

func TestValidatePath_PathTraversal(t *testing.T) {
	basedir := t.TempDir()

	_, err := ValidatePath(filepath.Join(basedir, "../../../etc/passwd"), []string{basedir})
	assert.Error(t, err)
}

func TestValidatePath_EmptyAllowedDirs(t *testing.T) {
	_, err := ValidatePath("/some/path", nil)
	assert.Error(t, err)
}

func TestValidatePath_OutsideAllowedDir(t *testing.T) {
	basedir := t.TempDir()
	otherDir := filepath.Join(t.TempDir(), "other")

	_, err := ValidatePath(otherDir, []string{basedir})
	assert.Error(t, err)
}

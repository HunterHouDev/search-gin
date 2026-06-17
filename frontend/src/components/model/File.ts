class FileModel {
  Id = '';
  Tags: [] = [];
  MovieType = '';
  FileType = '';
  Jpg = '';
  Png = '';
  Author = '';
  Code = '';
  MTime: Date | undefined;
  SizeStr: Date | undefined;
  Name = '';
  Title = '';
  Path = '';
  originName = '';

  fromObject(data: object) {
    Object.assign(this, data);
    return this;
  }
  isEmpty() {
    return this.Id == undefined || this.Id == null;
  }
}

class FileQuery extends FileModel {
  Page = 1;
  PageSize = 14;
  OnlyRepeat = false;
  Keyword = '';
  SortField = 'MTime';
  SortType = 'desc';
  showStyle = 'post';
  createdAt: Date | undefined;
}

export { FileQuery, FileModel };

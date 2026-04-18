package git

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	gogit "github.com/go-git/go-git/v5"
	gogitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

// goGitBackend 基于 github.com/go-git/go-git/v5 实现 Backend，
// 完全用纯 Go 代码完成 clone/pull/push/commit 等操作，无需本机安装 git。
type goGitBackend struct{}

func (goGitBackend) Name() string { return BackendGoGit }

func (goGitBackend) Clone(repoURL, path string) error {
	auth, _ := buildAuth(repoURL)
	_, err := gogit.PlainClone(path, false, &gogit.CloneOptions{
		URL:  repoURL,
		Auth: auth,
	})
	return err
}

func (goGitBackend) Init(path string) error {
	_, err := gogit.PlainInit(path, false)
	return err
}

// Pull 尽力模拟 exec 后端的语义：
//  1. 先尝试 fetch origin
//  2. 若本地还没有 HEAD（刚 init + remote add），则基于远程默认分支创建并 checkout 本地分支；
//  3. 若本地已有 HEAD，执行 Worktree.Pull（fast-forward 或合并，跟 exec 的 --rebase 结果接近）。
//
// 注意：go-git 原生不支持 rebase；对常见同步场景（本地工作区已清理再 pull）fast-forward 已足够。
func (goGitBackend) Pull(path string) error {
	r, err := gogit.PlainOpen(path)
	if err != nil {
		return err
	}

	auth, _ := buildAuth(goGitRemoteURL(r))

	if err := r.Fetch(&gogit.FetchOptions{RemoteName: "origin", Auth: auth}); err != nil {
		if !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
			return err
		}
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	if _, headErr := r.Head(); headErr != nil {
		branch := goGitDetectDefaultBranch(r)
		remoteBranchRef := plumbing.NewRemoteReferenceName("origin", branch)
		remoteRef, err := r.Reference(remoteBranchRef, true)
		if err != nil {
			return err
		}
		localRefName := plumbing.NewBranchReferenceName(branch)
		if err := r.Storer.SetReference(plumbing.NewHashReference(localRefName, remoteRef.Hash())); err != nil {
			return err
		}
		if err := r.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, localRefName)); err != nil {
			return err
		}
		_ = r.CreateBranch(&gogitconfig.Branch{
			Name:   branch,
			Remote: "origin",
			Merge:  localRefName,
		})
		return w.Checkout(&gogit.CheckoutOptions{Branch: localRefName, Force: true})
	}

	err = w.Pull(&gogit.PullOptions{RemoteName: "origin", Auth: auth})
	if err == nil || errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return nil
	}
	return err
}

func (goGitBackend) Commit(path, message string) error {
	r, err := gogit.PlainOpen(path)
	if err != nil {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	if err := w.AddWithOptions(&gogit.AddOptions{All: true}); err != nil {
		return err
	}
	sig := goGitSignature(r)
	_, err = w.Commit(message, &gogit.CommitOptions{Author: sig, Committer: sig})
	return err
}

func (goGitBackend) Push(path string) error {
	r, err := gogit.PlainOpen(path)
	if err != nil {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	_ = w.AddWithOptions(&gogit.AddOptions{All: true})
	status, _ := w.Status()
	if !status.IsClean() {
		sig := goGitSignature(r)
		_, _ = w.Commit("sync vault", &gogit.CommitOptions{Author: sig, Committer: sig})
	}
	auth, _ := buildAuth(goGitRemoteURL(r))
	err = r.Push(&gogit.PushOptions{RemoteName: "origin", Auth: auth})
	if err == nil || errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return nil
	}
	return err
}

func (goGitBackend) HasChanges(path string) bool {
	r, err := gogit.PlainOpen(path)
	if err != nil {
		return false
	}
	w, err := r.Worktree()
	if err != nil {
		return false
	}
	status, err := w.Status()
	if err != nil {
		return false
	}
	return !status.IsClean()
}

func (goGitBackend) AddRemote(path, repoURL string) error {
	r, err := gogit.PlainOpen(path)
	if err != nil {
		return err
	}
	_, err = r.CreateRemote(&gogitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})
	if err == nil {
		return nil
	}
	if errors.Is(err, gogit.ErrRemoteExists) {
		cfg, cerr := r.Config()
		if cerr != nil {
			return cerr
		}
		if remote, ok := cfg.Remotes["origin"]; ok {
			remote.URLs = []string{repoURL}
			return r.SetConfig(cfg)
		}
	}
	return err
}

func (goGitBackend) IsGitRepo(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

func (goGitBackend) RemoteHasCommit(path string) (bool, error) {
	r, err := gogit.PlainOpen(path)
	if err != nil {
		return false, err
	}
	remote, err := r.Remote("origin")
	if err != nil {
		return false, err
	}
	auth, _ := buildAuth(goGitRemoteURL(r))
	refs, err := remote.List(&gogit.ListOptions{Auth: auth})
	if err != nil {
		// 远程存在但里头空（无任何引用）→ 等同于 "没有提交"
		if errors.Is(err, transport.ErrEmptyRemoteRepository) {
			return false, nil
		}
		return false, err
	}
	for _, ref := range refs {
		if ref.Name().IsBranch() {
			return true, nil
		}
	}
	return false, nil
}

func (goGitBackend) HasOriginRemote(path string) (bool, error) {
	r, err := gogit.PlainOpen(path)
	if err != nil {
		return false, nil
	}
	if _, err := r.Remote("origin"); err != nil {
		return false, nil
	}
	return true, nil
}

// RestoreFile 读取 HEAD 上该文件的内容并写回工作区。
// 该实现仅恢复单个文件，不碰其他修改（与 exec 的 `git checkout -- <file>` 语义一致）。
func (goGitBackend) RestoreFile(repoRoot, fileName string) {
	r, err := gogit.PlainOpen(repoRoot)
	if err != nil {
		return
	}
	ref, err := r.Head()
	if err != nil {
		return
	}
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return
	}
	f, err := commit.File(fileName)
	if err != nil {
		return
	}
	reader, err := f.Reader()
	if err != nil {
		return
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(repoRoot, fileName), data, 0644)
}

func (goGitBackend) CurrentBranch(path string) string {
	r, err := gogit.PlainOpen(path)
	if err != nil {
		return ""
	}
	ref, err := r.Head()
	if err != nil {
		return ""
	}
	if ref.Name().IsBranch() {
		return ref.Name().Short()
	}
	return ""
}

func (goGitBackend) RemoteURL(path string) string {
	r, err := gogit.PlainOpen(path)
	if err != nil {
		return ""
	}
	remote, err := r.Remote("origin")
	if err != nil {
		return ""
	}
	urls := remote.Config().URLs
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// goGitRemoteURL 从一个已打开的仓库里取 origin 的 URL；
// 没有或读取失败时返回空串（buildAuth 收到空串会判定为非 SSH 直接返回 nil）。
func goGitRemoteURL(r *gogit.Repository) string {
	if r == nil {
		return ""
	}
	remote, err := r.Remote("origin")
	if err != nil {
		return ""
	}
	urls := remote.Config().URLs
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

func goGitDetectDefaultBranch(r *gogit.Repository) string {
	refs, err := r.References()
	if err != nil {
		return "main"
	}
	hasMain, hasMaster := false, false
	_ = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsRemote() {
			s := ref.Name().Short()
			if s == "origin/main" {
				hasMain = true
			}
			if s == "origin/master" {
				hasMaster = true
			}
		}
		return nil
	})
	if hasMain {
		return "main"
	}
	if hasMaster {
		return "master"
	}
	return "main"
}

// goGitSignature 优先使用仓库本地 config 的 user.name / user.email；
// 读不到时尝试用户 global gitconfig；再不行退回占位符，保证 commit 总能成功。
func goGitSignature(r *gogit.Repository) *object.Signature {
	name := ""
	email := ""

	if cfg, err := r.Config(); err == nil && cfg != nil {
		name = cfg.User.Name
		email = cfg.User.Email
	}

	if name == "" || email == "" {
		if g, err := gogitconfig.LoadConfig(gogitconfig.GlobalScope); err == nil && g != nil {
			if name == "" {
				name = g.User.Name
			}
			if email == "" {
				email = g.User.Email
			}
		}
	}

	if name == "" {
		name = "pwdmgr"
	}
	if email == "" {
		email = "pwdmgr@local"
	}

	return &object.Signature{
		Name:  name,
		Email: email,
		When:  time.Now(),
	}
}

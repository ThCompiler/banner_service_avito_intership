package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"bannersrv/internal/app/config"
	bp "bannersrv/internal/banner/repository/postgres"
	"bannersrv/internal/pkg/types"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tidwall/randjson"
)

type CorrectPair struct {
	BannerID  types.ID   `json:"banner_id"`
	FeatureID types.ID   `json:"feature_id"`
	TagIDs    []types.ID `json:"tag_ids"`
}

func main() {
	var featuresCount, tagsCount, countBanners uint64
	var configPath string

	flag.StringVar(&configPath, "config", "./config/localhost-config.yaml", "путь к конфигу подключения")
	flag.Uint64Var(&featuresCount, "features", 2000, "число фичей")
	flag.Uint64Var(&tagsCount, "tags", 2000, "число тэгов")
	flag.Uint64Var(&countBanners, "banners", 16000, "число баннеров")
	flag.Parse()

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	// Postgres
	cfx, err := pgxpool.ParseConfig(cfg.Postgres.URL)
	if err != nil {
		log.Fatalf("postgres.New: %s", err)
	}

	cfx.MaxConns = int32(cfg.Postgres.MaxConnections)
	cfx.MinConns = int32(cfg.Postgres.MinConnections)
	cfx.MaxConnIdleTime = time.Duration(cfg.Postgres.TTLIDleConnections) * time.Millisecond

	pg, err := pgxpool.NewWithConfig(context.Background(), cfx)
	if err != nil {
		log.Fatalf("postgres.New: %s", err)
	}
	defer pg.Close()

	if err = pg.Ping(context.Background()); err != nil {
		pg.Close()
		log.Fatalf("can't check connection to sql with error %s", err)
	}

	// Repository
	bannerRepository := bp.NewBannerRepository(pg)

	featureIDs := make([]types.ID, featuresCount)
	tagsIDs := make([]types.ID, tagsCount)

	for i := uint64(0); i < featuresCount; i++ {
		featureIDs[i] = types.ID(i + 1)
	}

	for i := uint64(0); i < tagsCount; i++ {
		tagsIDs[i] = types.ID(i + 1)
	}

	countTagsInBanner := tagsCount / (countBanners / featuresCount)

	result := make([]CorrectPair, 0)

	for i := uint64(0); i < featuresCount; i++ {
		for j := uint64(0); j < tagsCount; j += countTagsInBanner {
			tags := tagsIDs[j:min(j+countTagsInBanner, tagsCount)]
			featureID := featureIDs[i]
			banner := randjson.Make(12, nil)

			createdID, err := bannerRepository.CreateBanner(featureID, tags, types.Content(banner), true)
			if err != nil {
				log.Fatal(err)
			}

			result = append(result, CorrectPair{
				BannerID:  createdID,
				FeatureID: featureID,
				TagIDs:    tags,
			})
		}
	}

	file, err := os.OpenFile("./info.json", os.O_WRONLY|os.O_CREATE, 0o777)
	if err != nil {
		log.Fatal(err)
	}

	res, err := json.Marshal(result)
	if err != nil {
		log.Fatal(err)
	}

	_, err = file.Write(res)
	if err != nil {
		log.Fatal(err)
	}

	if err := file.Close(); err != nil {
		log.Fatal(err)
	}
}
